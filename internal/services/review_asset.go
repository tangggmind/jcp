package services

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/run-bigpig/jcp/internal/models"
)

const maxReviewImageBytes = 10 * 1024 * 1024

var reviewImageExtensions = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

func (s *ReviewService) SavePastedImage(req models.SaveReviewImageRequest) (models.SaveReviewImageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	article, err := s.getArticleByIDNoLock(req.ArticleID)
	if err != nil {
		return models.SaveReviewImageResult{}, err
	}
	mimeType := strings.ToLower(strings.TrimSpace(req.MimeType))
	if _, ok := reviewImageExtensions[mimeType]; !ok {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片格式不支持")
	}

	rawBase64 := strings.TrimSpace(req.DataBase64)
	if comma := strings.Index(rawBase64, ","); comma >= 0 {
		rawBase64 = rawBase64[comma+1:]
	}
	data, err := base64.StdEncoding.DecodeString(rawBase64)
	if err != nil {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片 base64 解析失败: %w", err)
	}
	if len(data) > maxReviewImageBytes {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片超过大小限制")
	}
	if detected := http.DetectContentType(data); detected != mimeType {
		if !(mimeType == "image/jpeg" && detected == "image/jpeg") {
			return models.SaveReviewImageResult{}, fmt.Errorf("图片格式不支持")
		}
	}

	return s.saveReviewImageDataNoLock(article, req.Date, mimeType, data, models.ReviewSourcePaste, "")
}

func (s *ReviewService) DownloadImage(req models.DownloadReviewImageRequest) (models.SaveReviewImageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	article, err := s.getArticleByIDNoLock(req.ArticleID)
	if err != nil {
		return models.SaveReviewImageResult{}, err
	}
	parsedURL, err := url.Parse(strings.TrimSpace(req.URL))
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片链接不是有效的 http/https 地址")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(parsedURL.String())
	if err != nil {
		return models.SaveReviewImageResult{}, fmt.Errorf("网络图片下载失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.SaveReviewImageResult{}, fmt.Errorf("网络图片下载失败: HTTP %d", resp.StatusCode)
	}

	contentType := strings.ToLower(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if _, ok := reviewImageExtensions[contentType]; !ok {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片格式不支持")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReviewImageBytes+1))
	if err != nil {
		return models.SaveReviewImageResult{}, fmt.Errorf("网络图片读取失败: %w", err)
	}
	if len(data) > maxReviewImageBytes {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片超过大小限制")
	}
	if detected := http.DetectContentType(data); detected != contentType {
		return models.SaveReviewImageResult{}, fmt.Errorf("图片格式不支持")
	}
	return s.saveReviewImageDataNoLock(article, req.Date, contentType, data, models.ReviewSourceDownload, parsedURL.String())
}

func (s *ReviewService) GetReviewAssetBase64(filePath string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	target, normalized, err := ResolveReviewSafePath(s.dataDir, filePath)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(normalized, "pictures/") {
		return "", fmt.Errorf("只能读取 pictures 目录下的复盘图片")
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("读取复盘图片失败: %w", err)
	}
	mimeType := http.DetectContentType(data)
	if _, ok := reviewImageExtensions[mimeType]; !ok {
		return "", fmt.Errorf("图片格式不支持")
	}
	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func (s *ReviewService) saveReviewImageDataNoLock(article models.ReviewArticle, requestedDate string, mimeType string, data []byte, source string, originalURL string) (models.SaveReviewImageResult, error) {
	ext := reviewImageExtensions[mimeType]
	hash := sha256.Sum256(data)
	hashText := hex.EncodeToString(hash[:])
	dirParts := []string{"pictures"}
	dateForName := article.Date
	if article.Type == models.ReviewTypeSummary {
		dirParts = append(dirParts, "summary")
		dateForName = "summary"
	} else {
		if requestedDate != "" {
			dateForName = requestedDate
		}
		parsed, err := time.Parse("2006-01-02", dateForName)
		if err != nil {
			return models.SaveReviewImageResult{}, fmt.Errorf("复盘日期格式错误，请使用 YYYY-MM-DD")
		}
		dirParts = append(dirParts, parsed.Format("2006"), parsed.Format("01"), parsed.Format("02"))
	}

	fileName := strings.ReplaceAll(dateForName, "-", "") + "-" + hashText[:12] + ext
	relativePath := filepath.ToSlash(filepath.Join(append(dirParts, fileName)...))
	target, filePath, err := ResolveReviewSafePath(s.dataDir, relativePath)
	if err != nil {
		return models.SaveReviewImageResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return models.SaveReviewImageResult{}, fmt.Errorf("创建图片目录失败: %w", err)
	}
	if _, err := os.Stat(target); os.IsNotExist(err) {
		if err := AtomicWriteReviewFile(target, data); err != nil {
			return models.SaveReviewImageResult{}, err
		}
	} else if err != nil {
		return models.SaveReviewImageResult{}, fmt.Errorf("检查图片文件失败: %w", err)
	}

	assetID := "asset-" + hashText[:16]
	now := time.Now().UnixMilli()
	if _, err := s.db.Exec(`
		INSERT INTO review_assets (id, article_id, source, original_url, file_path, mime_type, size, created_at, updated_at)
		VALUES (?, ?, ?, '', ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			article_id = excluded.article_id,
			file_path = excluded.file_path,
			mime_type = excluded.mime_type,
			size = excluded.size,
			updated_at = excluded.updated_at
	`, assetID, article.ID, source, filePath, mimeType, int64(len(data)), now, now); err != nil {
		return models.SaveReviewImageResult{}, fmt.Errorf("保存图片索引失败: %w", err)
	}
	if originalURL != "" {
		_, _ = s.db.Exec(`UPDATE review_assets SET original_url = ? WHERE id = ?`, originalURL, assetID)
	}

	markdownPath := "../" + filePath
	return models.SaveReviewImageResult{
		AssetID:      assetID,
		FilePath:     filePath,
		MarkdownPath: markdownPath,
		MarkdownText: fmt.Sprintf("![图片](%s)", markdownPath),
		MimeType:     mimeType,
		Size:         int64(len(data)),
	}, nil
}
