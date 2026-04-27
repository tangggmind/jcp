package services

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/run-bigpig/jcp/internal/models"
)

var tinyPNGBase64 = base64.StdEncoding.EncodeToString([]byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x02, 0x00, 0x00, 0x00,
})

func TestSavePastedImageStoresDailyImage(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()
	article, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}

	result, err := service.SavePastedImage(models.SaveReviewImageRequest{
		ArticleID:  article.Article.ID,
		Date:       "2026-04-27",
		FileName:   "ignored.png",
		MimeType:   "image/png",
		DataBase64: tinyPNGBase64,
	})
	if err != nil {
		t.Fatalf("SavePastedImage returned error: %v", err)
	}
	if !strings.HasPrefix(result.FilePath, "pictures/2026/04/27/") {
		t.Fatalf("unexpected file path: %s", result.FilePath)
	}
	if !strings.HasPrefix(result.MarkdownPath, "../pictures/2026/04/27/") || !strings.Contains(result.MarkdownText, result.MarkdownPath) {
		t.Fatalf("unexpected markdown result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(result.FilePath))); err != nil {
		t.Fatalf("expected image file: %v", err)
	}
}

func TestDownloadImageStoresHTTPImage(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()
	article, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}
	imageBytes, _ := base64.StdEncoding.DecodeString(tinyPNGBase64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(imageBytes)
	}))
	defer server.Close()

	result, err := service.DownloadImage(models.DownloadReviewImageRequest{
		ArticleID: article.Article.ID,
		Date:      "2026-04-27",
		URL:       server.URL + "/a.png",
	})
	if err != nil {
		t.Fatalf("DownloadImage returned error: %v", err)
	}
	if !strings.HasPrefix(result.FilePath, "pictures/2026/04/27/") {
		t.Fatalf("unexpected file path: %s", result.FilePath)
	}
	if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(result.FilePath))); err != nil {
		t.Fatalf("expected downloaded image file: %v", err)
	}
}

func TestDownloadImageRejectsInvalidSchemeAndFakeContent(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if _, err := service.DownloadImage(models.DownloadReviewImageRequest{ArticleID: "summary", URL: "file:///tmp/a.png"}); err == nil || !strings.Contains(err.Error(), "图片链接不是有效的 http/https 地址") {
		t.Fatalf("expected invalid scheme error, got %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("not an image"))
	}))
	defer server.Close()
	if _, err := service.DownloadImage(models.DownloadReviewImageRequest{ArticleID: "summary", URL: server.URL}); err == nil || !strings.Contains(err.Error(), "图片格式不支持") {
		t.Fatalf("expected fake content error, got %v", err)
	}
}

func TestGetReviewAssetBase64ReadsOnlyPictures(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()
	result, err := service.SavePastedImage(models.SaveReviewImageRequest{
		ArticleID:  "summary",
		MimeType:   "image/png",
		DataBase64: tinyPNGBase64,
	})
	if err != nil {
		t.Fatalf("SavePastedImage returned error: %v", err)
	}

	dataURL, err := service.GetReviewAssetBase64(result.FilePath)
	if err != nil {
		t.Fatalf("GetReviewAssetBase64 returned error: %v", err)
	}
	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("expected data url, got %s", dataURL)
	}

	if _, err := service.GetReviewAssetBase64("../articles/a.md"); err == nil || !strings.Contains(err.Error(), "复盘路径非法") {
		t.Fatalf("expected traversal error, got %v", err)
	}
	if _, err := service.GetReviewAssetBase64("articles/复盘总结.md"); err == nil || !strings.Contains(err.Error(), "只能读取 pictures") {
		t.Fatalf("expected non-pictures error, got %v", err)
	}
}

func TestSavePastedImageStoresSummaryImageAndRejectsInvalidInput(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	result, err := service.SavePastedImage(models.SaveReviewImageRequest{
		ArticleID:  "summary",
		FileName:   "summary.png",
		MimeType:   "image/png",
		DataBase64: "data:image/png;base64," + tinyPNGBase64,
	})
	if err != nil {
		t.Fatalf("SavePastedImage returned error: %v", err)
	}
	if !strings.HasPrefix(result.FilePath, "pictures/summary/") {
		t.Fatalf("summary image should be stored in summary dir, got %s", result.FilePath)
	}

	if _, err := service.SavePastedImage(models.SaveReviewImageRequest{ArticleID: "summary", MimeType: "text/plain", DataBase64: tinyPNGBase64}); err == nil || !strings.Contains(err.Error(), "图片格式不支持") {
		t.Fatalf("expected unsupported mime error, got %v", err)
	}
	tooLarge := base64.StdEncoding.EncodeToString(make([]byte, maxReviewImageBytes+1))
	if _, err := service.SavePastedImage(models.SaveReviewImageRequest{ArticleID: "summary", MimeType: "image/png", DataBase64: tooLarge}); err == nil || !strings.Contains(err.Error(), "图片超过大小限制") {
		t.Fatalf("expected size limit error, got %v", err)
	}
}
