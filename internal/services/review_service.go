package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/glebarez/go-sqlite"

	"github.com/run-bigpig/jcp/internal/models"
)

type ReviewService struct {
	dataDir      string
	articlesDir  string
	picturesDir  string
	templatesDir string
	dbPath       string
	db           *sql.DB
	mu           sync.RWMutex
}

func NewReviewService(dataDir string) (*ReviewService, error) {
	if dataDir == "" {
		dataDir = filepath.Join(".", "data")
	}

	service := &ReviewService{
		dataDir:      dataDir,
		articlesDir:  filepath.Join(dataDir, "articles"),
		picturesDir:  filepath.Join(dataDir, "pictures"),
		templatesDir: filepath.Join(dataDir, "review_templates"),
		dbPath:       filepath.Join(dataDir, "review.db"),
	}

	if err := service.ensureDirectories(); err != nil {
		return nil, err
	}
	if err := service.openDatabase(); err != nil {
		return nil, err
	}
	if err := service.migrate(); err != nil {
		_ = service.Close()
		return nil, err
	}
	if err := service.ensureDefaultDailyTemplate(); err != nil {
		_ = service.Close()
		return nil, err
	}
	if err := service.ensureSummaryArticle(); err != nil {
		_ = service.Close()
		return nil, err
	}

	return service, nil
}

func (s *ReviewService) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *ReviewService) ensureDirectories() error {
	for _, dir := range []string{s.dataDir, s.articlesDir, s.picturesDir, s.templatesDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("初始化复盘目录失败: %w", err)
		}
	}
	return nil
}

func (s *ReviewService) openDatabase() error {
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return fmt.Errorf("打开复盘数据库失败: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL;`); err != nil {
		_ = db.Close()
		return fmt.Errorf("设置复盘数据库 WAL 失败: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		_ = db.Close()
		return fmt.Errorf("设置复盘数据库 busy_timeout 失败: %w", err)
	}
	s.db = db
	return nil
}

func (s *ReviewService) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS review_articles (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			date TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL,
			file_path TEXT NOT NULL UNIQUE,
			template_id TEXT NOT NULL DEFAULT '',
			template_name TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			tags_json TEXT NOT NULL DEFAULT '[]',
			stocks_json TEXT NOT NULL DEFAULT '[]',
			profit_loss REAL NOT NULL DEFAULT 0,
			emotion TEXT NOT NULL DEFAULT '',
			discipline_score INTEGER NULL,
			image_count INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_review_articles_daily_date
			ON review_articles(date)
			WHERE type = 'daily_review' AND date <> '';`,
		`CREATE INDEX IF NOT EXISTS idx_review_articles_type_date
			ON review_articles(type, date);`,
		`CREATE INDEX IF NOT EXISTS idx_review_articles_updated_at
			ON review_articles(updated_at DESC);`,
		`CREATE TABLE IF NOT EXISTS review_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL UNIQUE,
			is_builtin INTEGER NOT NULL DEFAULT 0,
			is_default INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_review_templates_default
			ON review_templates(is_default);`,
		`CREATE TABLE IF NOT EXISTS review_assets (
			id TEXT PRIMARY KEY,
			article_id TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL,
			original_url TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL UNIQUE,
			mime_type TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_review_assets_article_id
			ON review_assets(article_id);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("迁移复盘数据库失败: %w", err)
		}
	}
	return nil
}

func (s *ReviewService) ensureDefaultDailyTemplate() error {
	const templateID = "default-daily"
	templatePath := filepath.Join(s.templatesDir, templateID+".md")

	if _, err := os.Stat(templatePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("检查默认复盘模板失败: %w", err)
		}
		if err := os.WriteFile(templatePath, []byte(defaultDailyTemplateContent), 0644); err != nil {
			return fmt.Errorf("创建默认复盘模板失败: %w", err)
		}
	} else {
		content, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("读取默认复盘模板失败: %w", err)
		}
		if string(content) == legacyDefaultDailyTemplateContent {
			if err := os.WriteFile(templatePath, []byte(defaultDailyTemplateContent), 0644); err != nil {
				return fmt.Errorf("升级默认复盘模板失败: %w", err)
			}
		}
	}

	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("初始化默认复盘模板失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE review_templates SET is_default = 0 WHERE id <> ?`, templateID); err != nil {
		return fmt.Errorf("初始化默认复盘模板失败: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO review_templates (
			id, name, description, file_path, is_builtin, is_default, created_at, updated_at
		) VALUES (?, ?, ?, ?, 1, 1, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			file_path = excluded.file_path,
			is_builtin = 1,
			is_default = 1,
			updated_at = excluded.updated_at
	`, templateID, "默认每日复盘模板", "内置每日复盘模板", filepath.ToSlash(filepath.Join("review_templates", templateID+".md")), now, now); err != nil {
		return fmt.Errorf("初始化默认复盘模板失败: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("初始化默认复盘模板失败: %w", err)
	}
	return nil
}

func (s *ReviewService) ListTemplates() ([]models.ReviewTemplate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, name, description, file_path, is_builtin, is_default, created_at, updated_at
		FROM review_templates
		ORDER BY is_default DESC, is_builtin DESC, updated_at DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("查询复盘模板失败: %w", err)
	}
	defer rows.Close()

	var templates []models.ReviewTemplate
	for rows.Next() {
		var template models.ReviewTemplate
		var filePath string
		var isBuiltin, isDefault int
		if err := rows.Scan(
			&template.ID,
			&template.Name,
			&template.Description,
			&filePath,
			&isBuiltin,
			&isDefault,
			&template.CreatedAt,
			&template.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("读取复盘模板失败: %w", err)
		}
		content, err := os.ReadFile(filepath.Join(s.dataDir, filepath.FromSlash(filePath)))
		if err != nil {
			return nil, fmt.Errorf("读取复盘模板文件失败: %w", err)
		}
		template.Content = string(content)
		template.IsBuiltin = isBuiltin == 1
		template.IsDefault = isDefault == 1
		templates = append(templates, template)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("读取复盘模板失败: %w", err)
	}
	return templates, nil
}

func (s *ReviewService) ensureSummaryArticle() error {
	const summaryID = "summary"
	summaryPath := filepath.Join(s.articlesDir, "复盘总结.md")

	if _, err := os.Stat(summaryPath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("检查总复盘总结文章失败: %w", err)
		}
		if err := os.WriteFile(summaryPath, []byte(defaultSummaryArticleContent(time.Now())), 0644); err != nil {
			return fmt.Errorf("创建总复盘总结文章失败: %w", err)
		}
	}

	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		INSERT INTO review_articles (
			id, type, date, title, file_path, template_id, template_name, summary,
			tags_json, stocks_json, profit_loss, emotion, discipline_score,
			image_count, created_at, updated_at
		) VALUES (?, ?, '', ?, ?, '', '', ?, '[]', '[]', 0, '', NULL, 0, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			date = '',
			title = excluded.title,
			file_path = excluded.file_path,
			summary = excluded.summary,
			updated_at = excluded.updated_at
	`, summaryID, models.ReviewTypeSummary, "复盘总结", filepath.ToSlash(filepath.Join("articles", "复盘总结.md")), "长期交易经验沉淀", now, now)
	if err != nil {
		return fmt.Errorf("初始化总复盘总结索引失败: %w", err)
	}
	return nil
}

func (s *ReviewService) GetSummaryArticle() (models.ReviewArticleDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getArticleDetailNoLock("summary")
}

func (s *ReviewService) CreateDailyReview(req models.CreateDailyReviewRequest) (models.ReviewArticleDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	reviewDate, err := time.Parse("2006-01-02", req.Date)
	if err != nil || reviewDate.Format("2006-01-02") != req.Date {
		return models.ReviewArticleDetail{}, fmt.Errorf("复盘日期格式错误，请使用 YYYY-MM-DD")
	}

	var existingCount int
	if err := s.db.QueryRow(
		`SELECT COUNT(1) FROM review_articles WHERE type = ? AND date = ?`,
		models.ReviewTypeDaily,
		req.Date,
	).Scan(&existingCount); err != nil {
		return models.ReviewArticleDetail{}, fmt.Errorf("检查每日复盘是否存在失败: %w", err)
	}
	if existingCount > 0 {
		return models.ReviewArticleDetail{}, fmt.Errorf("该日期复盘已存在")
	}

	relativePath := filepath.ToSlash(filepath.Join("articles", req.Date+".md"))
	targetPath, markdownPath, err := ResolveReviewSafePath(s.dataDir, relativePath)
	if err != nil {
		return models.ReviewArticleDetail{}, err
	}
	if _, err := os.Stat(targetPath); err == nil {
		return models.ReviewArticleDetail{}, fmt.Errorf("该日期复盘已存在")
	} else if !os.IsNotExist(err) {
		return models.ReviewArticleDetail{}, fmt.Errorf("检查每日复盘文件失败: %w", err)
	}

	templateID := strings.TrimSpace(req.TemplateID)
	if templateID == "" {
		templateID = "default-daily"
	}
	template, err := s.getTemplateByIDNoLock(templateID)
	warning := ""
	if err != nil {
		if templateID == "default-daily" {
			return models.ReviewArticleDetail{}, err
		}
		warning = "指定模板不存在，已使用默认模板"
		template, err = s.getTemplateByIDNoLock("default-daily")
		if err != nil {
			return models.ReviewArticleDetail{}, err
		}
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = req.Date + " 每日复盘"
	}

	body := renderReviewTemplate(template.Content, map[string]string{
		"date":           req.Date,
		"weekday":        chineseWeekday(reviewDate.Weekday()),
		"title":          title,
		"selectedStocks": strings.Join(req.Stocks, "、"),
		"marketSnapshot": strings.TrimSpace(req.MarketSnapshot),
	})
	now := time.Now()
	article := models.ReviewArticle{
		ID:           "daily-" + req.Date,
		Type:         models.ReviewTypeDaily,
		Date:         req.Date,
		Title:        title,
		FilePath:     markdownPath,
		TemplateID:   template.ID,
		TemplateName: template.Name,
		Tags:         []string{},
		Stocks:       normalizeStringList(req.Stocks),
		ProfitLoss:   0,
		Emotion:      "",
		CreatedAt:    now.UnixMilli(),
		UpdatedAt:    now.UnixMilli(),
	}
	content := buildReviewArticleMarkdown(article, body, now)
	parsed := ParseReviewMarkdown(content)
	article.Summary = parsed.Summary
	article.ImageCount = len(parsed.ImageRefs)
	content = buildReviewArticleMarkdown(article, body, now)

	if err := AtomicWriteReviewFile(targetPath, []byte(content)); err != nil {
		return models.ReviewArticleDetail{}, err
	}
	if err := s.upsertArticleNoLock(article); err != nil {
		return models.ReviewArticleDetail{}, err
	}

	return models.ReviewArticleDetail{
		Article: article,
		Content: content,
		Warning: warning,
	}, nil
}

func (s *ReviewService) GetArticle(id string) (models.ReviewArticleDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if strings.TrimSpace(id) == "" {
		return models.ReviewArticleDetail{}, fmt.Errorf("文章 ID 不能为空")
	}
	return s.getArticleDetailNoLock(id)
}

func (s *ReviewService) ListArticles(req models.ReviewListRequest) (models.ReviewListResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	rows, err := s.db.Query(`
		SELECT id, type, date, title, file_path, template_id, template_name, summary,
			tags_json, stocks_json, profit_loss, emotion, discipline_score,
			image_count, created_at, updated_at
		FROM review_articles
		ORDER BY
			CASE WHEN type = 'summary_review' THEN 0 ELSE 1 END ASC,
			date DESC,
			updated_at DESC
	`)
	if err != nil {
		return models.ReviewListResult{}, fmt.Errorf("查询复盘文章列表失败: %w", err)
	}
	defer rows.Close()

	var filtered []models.ReviewArticle
	for rows.Next() {
		article, err := scanReviewArticle(rows)
		if err != nil {
			return models.ReviewListResult{}, err
		}
		matched, err := s.articleMatchesListRequestNoLock(article, req)
		if err != nil {
			return models.ReviewListResult{}, err
		}
		if matched {
			filtered = append(filtered, article)
		}
	}
	if err := rows.Err(); err != nil {
		return models.ReviewListResult{}, fmt.Errorf("读取复盘文章列表失败: %w", err)
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return models.ReviewListResult{Items: []models.ReviewArticle{}, Total: total}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return models.ReviewListResult{Items: filtered[start:end], Total: total}, nil
}

func (s *ReviewService) SaveArticle(req models.SaveReviewArticleRequest) (models.ReviewArticleDetail, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(req.ID) == "" {
		return models.ReviewArticleDetail{}, fmt.Errorf("文章 ID 不能为空")
	}

	article, err := s.getArticleByIDNoLock(req.ID)
	if err != nil {
		return models.ReviewArticleDetail{}, err
	}

	parsed := ParseReviewMarkdown(req.Content)
	body := parsed.Content
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = strings.TrimSpace(parsed.Meta.Title)
	}
	if title == "" {
		title = extractFirstMarkdownHeading(body)
	}
	if title == "" {
		title = article.Title
	}

	if parsed.HasFrontMatter {
		article.Tags = parsed.Meta.Tags
		article.Stocks = parsed.Meta.Stocks
		article.ProfitLoss = parsed.Meta.ProfitLoss
		article.Emotion = parsed.Meta.Emotion
		article.DisciplineScore = parsed.Meta.DisciplineScore
	}
	article.Title = title
	article.Summary = parsed.Summary
	article.ImageCount = len(parsed.ImageRefs)
	article.UpdatedAt = time.Now().UnixMilli()

	now := time.Now()
	content := buildReviewArticleMarkdown(article, body, now)
	target, _, err := ResolveReviewSafePath(s.dataDir, article.FilePath)
	if err != nil {
		return models.ReviewArticleDetail{}, err
	}
	if err := AtomicWriteReviewFile(target, []byte(content)); err != nil {
		return models.ReviewArticleDetail{}, err
	}

	detail := models.ReviewArticleDetail{
		Article: article,
		Content: content,
		Warning: parsed.Warning,
	}
	if err := s.upsertArticleNoLock(article); err != nil {
		detail.Warning = strings.TrimSpace(strings.Join([]string{detail.Warning, "文章已保存，索引更新失败，可重建索引"}, " "))
		return detail, nil
	}
	return detail, nil
}

func (s *ReviewService) DeleteArticle(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("文章 ID 不能为空")
	}
	article, err := s.getArticleByIDNoLock(id)
	if err != nil {
		return err
	}
	if article.Type == models.ReviewTypeSummary {
		return fmt.Errorf("总复盘总结不可删除")
	}

	target, _, err := ResolveReviewSafePath(s.dataDir, article.FilePath)
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除复盘文章文件失败: %w", err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("删除复盘文章失败: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM review_articles WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除复盘文章索引失败: %w", err)
	}
	if _, err := tx.Exec(`UPDATE review_assets SET article_id = '' WHERE article_id = ?`, id); err != nil {
		return fmt.Errorf("解除复盘图片关联失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("删除复盘文章失败: %w", err)
	}
	return nil
}

func (s *ReviewService) RebuildIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.db.Exec(`DELETE FROM review_articles`); err != nil {
		return fmt.Errorf("清空复盘文章索引失败: %w", err)
	}
	if _, err := s.db.Exec(`UPDATE review_assets SET article_id = ''`); err != nil {
		return fmt.Errorf("重置复盘图片关联失败: %w", err)
	}

	entries, err := os.ReadDir(s.articlesDir)
	if err != nil {
		return fmt.Errorf("扫描复盘文章目录失败: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		article, refs, ok, err := s.articleFromMarkdownFileNoLock(entry.Name())
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := s.upsertArticleNoLock(article); err != nil {
			return err
		}
		for _, ref := range refs {
			if normalized := normalizeReviewImageRef(ref); normalized != "" {
				_, _ = s.db.Exec(`UPDATE review_assets SET article_id = ? WHERE file_path = ?`, article.ID, normalized)
			}
		}
	}
	return nil
}

func (s *ReviewService) getArticleByIDNoLock(id string) (models.ReviewArticle, error) {
	row := s.db.QueryRow(`
		SELECT id, type, date, title, file_path, template_id, template_name, summary,
			tags_json, stocks_json, profit_loss, emotion, discipline_score,
			image_count, created_at, updated_at
		FROM review_articles
		WHERE id = ?
	`, id)
	article, err := scanReviewArticle(row)
	if err == sql.ErrNoRows {
		return models.ReviewArticle{}, fmt.Errorf("文章不存在")
	}
	if err != nil {
		return models.ReviewArticle{}, err
	}
	return article, nil
}

func (s *ReviewService) articleFromMarkdownFileNoLock(name string) (models.ReviewArticle, []string, bool, error) {
	relativePath := filepath.ToSlash(filepath.Join("articles", name))
	target, markdownPath, err := ResolveReviewSafePath(s.dataDir, relativePath)
	if err != nil {
		return models.ReviewArticle{}, nil, false, err
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return models.ReviewArticle{}, nil, false, fmt.Errorf("读取复盘文章失败: %w", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		return models.ReviewArticle{}, nil, false, fmt.Errorf("读取复盘文章信息失败: %w", err)
	}

	base := strings.TrimSuffix(name, filepath.Ext(name))
	articleType := ""
	date := ""
	id := ""
	if name == "复盘总结.md" {
		articleType = models.ReviewTypeSummary
		id = "summary"
	} else if isValidReviewDate(base) {
		articleType = models.ReviewTypeDaily
		date = base
		id = "daily-" + base
	} else {
		return models.ReviewArticle{}, nil, false, nil
	}

	parsed := ParseReviewMarkdown(string(content))
	title := strings.TrimSpace(parsed.Meta.Title)
	if title == "" {
		title = extractFirstMarkdownHeading(parsed.Content)
	}
	if title == "" {
		if articleType == models.ReviewTypeSummary {
			title = "复盘总结"
		} else {
			title = date + " 每日复盘"
		}
	}

	article := models.ReviewArticle{
		ID:              id,
		Type:            articleType,
		Date:            date,
		Title:           title,
		FilePath:        markdownPath,
		TemplateID:      parsed.Meta.TemplateID,
		TemplateName:    parsed.Meta.TemplateName,
		Summary:         parsed.Summary,
		Tags:            parsed.Meta.Tags,
		Stocks:          parsed.Meta.Stocks,
		ProfitLoss:      parsed.Meta.ProfitLoss,
		Emotion:         parsed.Meta.Emotion,
		DisciplineScore: parsed.Meta.DisciplineScore,
		ImageCount:      len(parsed.ImageRefs),
		CreatedAt:       info.ModTime().UnixMilli(),
		UpdatedAt:       info.ModTime().UnixMilli(),
	}
	return article, parsed.ImageRefs, true, nil
}

func isValidReviewDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

func normalizeReviewImageRef(ref string) string {
	ref = strings.TrimSpace(strings.Trim(ref, `"'`))
	ref = strings.TrimPrefix(ref, "../")
	if strings.HasPrefix(ref, "pictures/") {
		return ref
	}
	return ""
}

type reviewArticleScanner interface {
	Scan(dest ...any) error
}

func scanReviewArticle(scanner reviewArticleScanner) (models.ReviewArticle, error) {
	var article models.ReviewArticle
	var tagsJSON, stocksJSON string
	var score sql.NullInt64
	err := scanner.Scan(
		&article.ID,
		&article.Type,
		&article.Date,
		&article.Title,
		&article.FilePath,
		&article.TemplateID,
		&article.TemplateName,
		&article.Summary,
		&tagsJSON,
		&stocksJSON,
		&article.ProfitLoss,
		&article.Emotion,
		&score,
		&article.ImageCount,
		&article.CreatedAt,
		&article.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return models.ReviewArticle{}, err
	}
	if err != nil {
		return models.ReviewArticle{}, fmt.Errorf("查询复盘文章失败: %w", err)
	}
	if err := decodeStringList(tagsJSON, &article.Tags); err != nil {
		return models.ReviewArticle{}, fmt.Errorf("解析文章标签失败: %w", err)
	}
	if err := decodeStringList(stocksJSON, &article.Stocks); err != nil {
		return models.ReviewArticle{}, fmt.Errorf("解析文章股票失败: %w", err)
	}
	if score.Valid {
		value := int(score.Int64)
		article.DisciplineScore = &value
	}
	return article, nil
}

func (s *ReviewService) getArticleDetailNoLock(id string) (models.ReviewArticleDetail, error) {
	article, err := s.getArticleByIDNoLock(id)
	if err != nil {
		return models.ReviewArticleDetail{}, err
	}
	content, err := s.readArticleContentNoLock(article)
	if err != nil {
		return models.ReviewArticleDetail{}, err
	}
	return models.ReviewArticleDetail{
		Article: article,
		Content: content,
	}, nil
}

func (s *ReviewService) readArticleContentNoLock(article models.ReviewArticle) (string, error) {
	target, _, err := ResolveReviewSafePath(s.dataDir, article.FilePath)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("读取复盘文章失败: %w", err)
	}
	return string(content), nil
}

func (s *ReviewService) articleMatchesListRequestNoLock(article models.ReviewArticle, req models.ReviewListRequest) (bool, error) {
	if req.Type != "" && article.Type != req.Type {
		return false, nil
	}
	if req.StartDate != "" && article.Date != "" && article.Date < req.StartDate {
		return false, nil
	}
	if req.EndDate != "" && article.Date != "" && article.Date > req.EndDate {
		return false, nil
	}
	if !containsAll(article.Tags, normalizeStringList(req.Tags)) {
		return false, nil
	}
	if !containsAll(article.Stocks, normalizeStringList(req.Stocks)) {
		return false, nil
	}

	query := strings.TrimSpace(req.Query)
	if query == "" {
		return true, nil
	}
	haystack := strings.Join([]string{
		article.Title,
		article.Summary,
		strings.Join(article.Tags, " "),
		strings.Join(article.Stocks, " "),
	}, " ")
	content, err := s.readArticleContentNoLock(article)
	if err == nil {
		haystack += " " + content
	}
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(query)), nil
}

func containsAll(values []string, required []string) bool {
	if len(required) == 0 {
		return true
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

func extractFirstMarkdownHeading(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

func (s *ReviewService) getTemplateByIDNoLock(id string) (models.ReviewTemplate, error) {
	var template models.ReviewTemplate
	var filePath string
	var isBuiltin, isDefault int
	err := s.db.QueryRow(`
		SELECT id, name, description, file_path, is_builtin, is_default, created_at, updated_at
		FROM review_templates
		WHERE id = ?
	`, id).Scan(
		&template.ID,
		&template.Name,
		&template.Description,
		&filePath,
		&isBuiltin,
		&isDefault,
		&template.CreatedAt,
		&template.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return models.ReviewTemplate{}, fmt.Errorf("模板不存在")
	}
	if err != nil {
		return models.ReviewTemplate{}, fmt.Errorf("查询复盘模板失败: %w", err)
	}
	content, err := os.ReadFile(filepath.Join(s.dataDir, filepath.FromSlash(filePath)))
	if err != nil {
		return models.ReviewTemplate{}, fmt.Errorf("读取复盘模板文件失败: %w", err)
	}
	template.Content = string(content)
	template.IsBuiltin = isBuiltin == 1
	template.IsDefault = isDefault == 1
	return template, nil
}

func (s *ReviewService) upsertArticleNoLock(article models.ReviewArticle) error {
	tagsJSON, err := json.Marshal(normalizeStringList(article.Tags))
	if err != nil {
		return fmt.Errorf("序列化文章标签失败: %w", err)
	}
	stocksJSON, err := json.Marshal(normalizeStringList(article.Stocks))
	if err != nil {
		return fmt.Errorf("序列化文章股票失败: %w", err)
	}

	var score any
	if article.DisciplineScore != nil {
		score = *article.DisciplineScore
	}

	_, err = s.db.Exec(`
		INSERT INTO review_articles (
			id, type, date, title, file_path, template_id, template_name, summary,
			tags_json, stocks_json, profit_loss, emotion, discipline_score,
			image_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = excluded.type,
			date = excluded.date,
			title = excluded.title,
			file_path = excluded.file_path,
			template_id = excluded.template_id,
			template_name = excluded.template_name,
			summary = excluded.summary,
			tags_json = excluded.tags_json,
			stocks_json = excluded.stocks_json,
			profit_loss = excluded.profit_loss,
			emotion = excluded.emotion,
			discipline_score = excluded.discipline_score,
			image_count = excluded.image_count,
			updated_at = excluded.updated_at
	`, article.ID, article.Type, article.Date, article.Title, article.FilePath, article.TemplateID,
		article.TemplateName, article.Summary, string(tagsJSON), string(stocksJSON), article.ProfitLoss,
		article.Emotion, score, article.ImageCount, article.CreatedAt, article.UpdatedAt)
	if err != nil {
		return fmt.Errorf("同步复盘文章索引失败: %w", err)
	}
	return nil
}

func buildReviewArticleMarkdown(article models.ReviewArticle, body string, timestamp time.Time) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("type: " + article.Type + "\n")
	if article.Date != "" {
		sb.WriteString("date: " + article.Date + "\n")
	}
	sb.WriteString("title: " + strconv.Quote(article.Title) + "\n")
	if article.TemplateID != "" {
		sb.WriteString("templateId: " + article.TemplateID + "\n")
		sb.WriteString("templateName: " + strconv.Quote(article.TemplateName) + "\n")
	}
	sb.WriteString("summary: " + strconv.Quote(article.Summary) + "\n")
	sb.WriteString("tags: " + formatYAMLStringList(article.Tags) + "\n")
	sb.WriteString("stocks: " + formatYAMLStringList(article.Stocks) + "\n")
	if article.Type == models.ReviewTypeDaily {
		sb.WriteString(fmt.Sprintf("profitLoss: %v\n", article.ProfitLoss))
		sb.WriteString("emotion: " + strconv.Quote(article.Emotion) + "\n")
		if article.DisciplineScore == nil {
			sb.WriteString("disciplineScore: null\n")
		} else {
			sb.WriteString(fmt.Sprintf("disciplineScore: %d\n", *article.DisciplineScore))
		}
	}
	formattedTime := timestamp.Format(time.RFC3339)
	sb.WriteString("createdAt: " + formattedTime + "\n")
	sb.WriteString("updatedAt: " + formattedTime + "\n")
	sb.WriteString("---\n\n")
	sb.WriteString(strings.TrimSpace(body))
	sb.WriteString("\n")
	return sb.String()
}

func formatYAMLStringList(values []string) string {
	values = normalizeStringList(values)
	if len(values) == 0 {
		return "[]"
	}
	quoted := make([]string, 0, len(values))
	for _, value := range values {
		quoted = append(quoted, strconv.Quote(value))
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func renderReviewTemplate(content string, values map[string]string) string {
	rendered := content
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return rendered
}

func chineseWeekday(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "星期一"
	case time.Tuesday:
		return "星期二"
	case time.Wednesday:
		return "星期三"
	case time.Thursday:
		return "星期四"
	case time.Friday:
		return "星期五"
	case time.Saturday:
		return "星期六"
	default:
		return "星期日"
	}
}

func decodeStringList(raw string, target *[]string) error {
	if raw == "" {
		*target = []string{}
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return err
	}
	if *target == nil {
		*target = []string{}
	}
	return nil
}

const legacyDefaultDailyTemplateContent = `# {{date}} 每日复盘

> 标题：{{title}}
> 星期：{{weekday}}

## 今日市场

- 上证：
- 深成指：
- 创业板：
- 市场情绪：

## 今日操作

| 股票 | 操作 | 价格 | 仓位 | 理由 | 结果 |
| ---- | ---- | ---- | ---- | ---- | ---- |

## 持仓复盘

| 股票 | 当前判断 | 风险点 | 下一步计划 |
| ---- | -------- | ------ | ---------- |

## 错误与偏差

- 

## 做得好的地方

- 

## 情绪记录

- 

## 明日计划

- 
`

const defaultDailyTemplateContent = `# {{date}} 每日复盘

> 标题：{{title}}
> 星期：{{weekday}}

## 今日市场

市场概览：{{marketSnapshot}}
- 市场情绪：

## 今日操作

| 股票 | 操作 | 价格 | 仓位 | 理由 | 结果 |
| ---- | ---- | ---- | ---- | ---- | ---- |

## 持仓复盘

| 股票 | 当前判断 | 风险点 | 下一步计划 |
| ---- | -------- | ------ | ---------- |

## 错误与偏差

- 

## 做得好的地方

- 

## 情绪记录

- 

## 明日计划

- 
`

func defaultSummaryArticleContent(now time.Time) string {
	timestamp := now.Format(time.RFC3339)
	return fmt.Sprintf(`---
type: summary_review
title: 复盘总结
summary: 长期交易经验沉淀
tags: []
stocks: []
createdAt: %s
updatedAt: %s
---

# 复盘总结

## 高频错误

## 有效策略

## 交易纪律

## 仓位管理

## 情绪管理

## 可复用检查清单
`, timestamp, timestamp)
}
