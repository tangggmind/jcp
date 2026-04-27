package services

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/run-bigpig/jcp/internal/models"
)

func TestNewReviewServiceCreatesDirectoriesAndDatabase(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	for _, dir := range []string{"articles", "pictures", "review_templates"} {
		path := filepath.Join(dataDir, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected directory %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be a directory", dir)
		}
	}

	if _, err := os.Stat(filepath.Join(dataDir, "review.db")); err != nil {
		t.Fatalf("expected review.db: %v", err)
	}
}

func TestNewReviewServiceMigratesSchema(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	for _, name := range []string{"review_articles", "review_templates", "review_assets"} {
		if !sqliteObjectExists(t, service.db, "table", name) {
			t.Fatalf("expected table %s to exist", name)
		}
	}

	for _, name := range []string{
		"idx_review_articles_type_date",
		"idx_review_articles_updated_at",
		"idx_review_assets_article_id",
		"idx_review_templates_default",
	} {
		if !sqliteObjectExists(t, service.db, "index", name) {
			t.Fatalf("expected index %s to exist", name)
		}
	}
}

func TestNewReviewServiceIsIdempotent(t *testing.T) {
	dataDir := t.TempDir()

	first, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("first init returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close first service: %v", err)
	}

	second, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("second init returned error: %v", err)
	}
	defer second.Close()

	if !sqliteObjectExists(t, second.db, "table", "review_articles") {
		t.Fatalf("expected schema to remain available after repeated init")
	}
}

func TestNewReviewServiceReturnsChineseErrorForInvalidDataDir(t *testing.T) {
	dataDir := t.TempDir()
	filePath := filepath.Join(dataDir, "not-a-directory")
	if err := os.WriteFile(filePath, []byte("occupied"), 0644); err != nil {
		t.Fatalf("write occupied file: %v", err)
	}

	service, err := NewReviewService(filePath)
	if err == nil {
		service.Close()
		t.Fatalf("expected error for file dataDir")
	}
	if !strings.Contains(err.Error(), "初始化复盘目录失败") {
		t.Fatalf("expected Chinese init error, got %v", err)
	}
}

func TestNewReviewServiceEnsuresDefaultDailyTemplate(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	templatePath := filepath.Join(dataDir, "review_templates", "default-daily.md")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("expected default template file: %v", err)
	}

	for _, section := range []string{"今日市场", "今日操作", "持仓复盘", "错误与偏差", "情绪记录", "明日计划"} {
		if !strings.Contains(string(content), section) {
			t.Fatalf("default template should contain section %q", section)
		}
	}
	for _, variable := range []string{"{{date}}", "{{weekday}}", "{{title}}"} {
		if !strings.Contains(string(content), variable) {
			t.Fatalf("default template should keep variable %q", variable)
		}
	}
	if !strings.Contains(string(content), "{{marketSnapshot}}") {
		t.Fatalf("default template should keep marketSnapshot variable")
	}
	for _, removed := range []string{"- 上证：", "- 深成指：", "- 创业板："} {
		if strings.Contains(string(content), removed) {
			t.Fatalf("default template should not contain legacy index placeholder %q", removed)
		}
	}

	templates, err := service.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected one template, got %d", len(templates))
	}
	if templates[0].ID != "default-daily" || !templates[0].IsBuiltin || !templates[0].IsDefault {
		t.Fatalf("unexpected default template metadata: %+v", templates[0])
	}

	if countDefaultTemplates(t, service.db) != 1 {
		t.Fatalf("expected exactly one default template")
	}
}

func TestNewReviewServiceUpgradesLegacyDefaultTemplateFile(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	templatePath := filepath.Join(dataDir, "review_templates", "default-daily.md")
	if err := os.WriteFile(templatePath, []byte(legacyDefaultDailyTemplateContent), 0644); err != nil {
		t.Fatalf("write legacy template: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close service: %v", err)
	}

	service, err = NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("reinit service: %v", err)
	}
	defer service.Close()

	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read upgraded template: %v", err)
	}
	if !strings.Contains(string(content), "{{marketSnapshot}}") {
		t.Fatalf("legacy default template should be upgraded, got %s", content)
	}
	if strings.Contains(string(content), "- 上证：") {
		t.Fatalf("upgraded template should remove legacy index placeholders, got %s", content)
	}
}

func TestNewReviewServiceRepairsDefaultTemplateFile(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	templatePath := filepath.Join(dataDir, "review_templates", "default-daily.md")
	if err := os.Remove(templatePath); err != nil {
		t.Fatalf("remove default template file: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close service: %v", err)
	}

	service, err = NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("reinit service: %v", err)
	}
	defer service.Close()

	if _, err := os.Stat(templatePath); err != nil {
		t.Fatalf("expected repaired default template file: %v", err)
	}
	if countDefaultTemplates(t, service.db) != 1 {
		t.Fatalf("expected exactly one default template after repair")
	}
}

func TestNewReviewServiceRepairsDefaultTemplateRecordWithoutOverwritingFile(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	templatePath := filepath.Join(dataDir, "review_templates", "default-daily.md")
	customContent := "# 用户修改后的模板\n\n## 今日市场\n\n保留这行"
	if err := os.WriteFile(templatePath, []byte(customContent), 0644); err != nil {
		t.Fatalf("write custom template: %v", err)
	}
	if _, err := service.db.Exec(`DELETE FROM review_templates WHERE id = ?`, "default-daily"); err != nil {
		t.Fatalf("delete default template row: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close service: %v", err)
	}

	service, err = NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("reinit service: %v", err)
	}
	defer service.Close()

	content, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read default template file: %v", err)
	}
	if string(content) != customContent {
		t.Fatalf("default template file should not be overwritten, got %q", string(content))
	}

	templates, err := service.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected repaired template record, got %d", len(templates))
	}
}

func TestNewReviewServiceEnsuresSummaryArticle(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	summaryPath := filepath.Join(dataDir, "articles", "复盘总结.md")
	content, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("expected summary article file: %v", err)
	}

	for _, expected := range []string{
		"type: summary_review",
		"title: 复盘总结",
		"高频错误",
		"有效策略",
		"交易纪律",
		"仓位管理",
		"情绪管理",
		"可复用检查清单",
	} {
		if !strings.Contains(string(content), expected) {
			t.Fatalf("summary article should contain %q", expected)
		}
	}

	detail, err := service.GetSummaryArticle()
	if err != nil {
		t.Fatalf("GetSummaryArticle returned error: %v", err)
	}
	if detail.Article.ID != "summary" {
		t.Fatalf("unexpected summary id: %s", detail.Article.ID)
	}
	if detail.Article.Type != models.ReviewTypeSummary || detail.Article.Date != "" {
		t.Fatalf("unexpected summary article metadata: %+v", detail.Article)
	}
	if detail.Article.Title != "复盘总结" || detail.Article.FilePath != "articles/复盘总结.md" {
		t.Fatalf("unexpected summary article path/title: %+v", detail.Article)
	}
}

func TestNewReviewServiceDoesNotOverwriteEditedSummaryArticle(t *testing.T) {
	dataDir := t.TempDir()

	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	summaryPath := filepath.Join(dataDir, "articles", "复盘总结.md")
	customContent := "# 复盘总结\n\n用户长期沉淀内容"
	if err := os.WriteFile(summaryPath, []byte(customContent), 0644); err != nil {
		t.Fatalf("write custom summary: %v", err)
	}
	if err := service.Close(); err != nil {
		t.Fatalf("close service: %v", err)
	}

	service, err = NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("reinit service: %v", err)
	}
	defer service.Close()

	content, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read summary article: %v", err)
	}
	if string(content) != customContent {
		t.Fatalf("summary article should not be overwritten, got %q", string(content))
	}

	detail, err := service.GetSummaryArticle()
	if err != nil {
		t.Fatalf("GetSummaryArticle returned error: %v", err)
	}
	if detail.Article.Type != models.ReviewTypeSummary {
		t.Fatalf("summary article should be inferred as summary_review: %+v", detail.Article)
	}
}

func sqliteObjectExists(t *testing.T, db *sql.DB, objectType, name string) bool {
	t.Helper()

	var count int
	err := db.QueryRow(
		`SELECT COUNT(1) FROM sqlite_master WHERE type = ? AND name = ?`,
		objectType,
		name,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master for %s %s: %v", objectType, name, err)
	}
	return count > 0
}

func countDefaultTemplates(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM review_templates WHERE is_default = 1`).Scan(&count); err != nil {
		t.Fatalf("count default templates: %v", err)
	}
	return count
}

func requireTemplate(t *testing.T, templates []models.ReviewTemplate, id string) models.ReviewTemplate {
	t.Helper()

	for _, template := range templates {
		if template.ID == id {
			return template
		}
	}
	t.Fatalf("template %s not found in %+v", id, templates)
	return models.ReviewTemplate{}
}
