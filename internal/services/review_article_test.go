package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/run-bigpig/jcp/internal/models"
)

func TestCreateDailyReviewCreatesMarkdownAndIndex(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	detail, err := service.CreateDailyReview(models.CreateDailyReviewRequest{
		Date:       "2026-04-27",
		TemplateID: "default-daily",
		Stocks:     []string{"600000"},
	})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}

	if detail.Article.ID != "daily-2026-04-27" {
		t.Fatalf("unexpected article id: %s", detail.Article.ID)
	}
	if detail.Article.Type != models.ReviewTypeDaily || detail.Article.Date != "2026-04-27" {
		t.Fatalf("unexpected article metadata: %+v", detail.Article)
	}
	if detail.Article.FilePath != "articles/2026-04-27.md" {
		t.Fatalf("unexpected filePath: %s", detail.Article.FilePath)
	}
	if detail.Article.TemplateID != "default-daily" {
		t.Fatalf("unexpected template id: %s", detail.Article.TemplateID)
	}

	content, err := os.ReadFile(filepath.Join(dataDir, "articles", "2026-04-27.md"))
	if err != nil {
		t.Fatalf("expected article file: %v", err)
	}
	for _, expected := range []string{
		"type: daily_review",
		"date: 2026-04-27",
		"templateId: default-daily",
		"# 2026-04-27 每日复盘",
		"今日市场",
		"市场概览：",
		"600000",
	} {
		if !strings.Contains(string(content), expected) {
			t.Fatalf("article content should contain %q", expected)
		}
	}
	for _, removed := range []string{"- 上证：", "- 深成指：", "- 创业板："} {
		if strings.Contains(string(content), removed) {
			t.Fatalf("article content should not contain legacy index placeholder %q", removed)
		}
	}
}

func TestCreateDailyReviewRendersTemplateVariables(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	detail, err := service.CreateDailyReview(models.CreateDailyReviewRequest{
		Date:       "2026-04-27",
		TemplateID: "default-daily",
		Title:      "周一交易复盘",
	})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}
	if strings.Contains(detail.Content, "{{date}}") ||
		strings.Contains(detail.Content, "{{weekday}}") ||
		strings.Contains(detail.Content, "{{title}}") {
		t.Fatalf("template variables should be rendered, got %s", detail.Content)
	}
	if !strings.Contains(detail.Content, "星期一") || !strings.Contains(detail.Content, "周一交易复盘") {
		t.Fatalf("rendered content should include weekday and title, got %s", detail.Content)
	}
}

func TestCreateDailyReviewRendersMarketSnapshot(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	detail, err := service.CreateDailyReview(models.CreateDailyReviewRequest{
		Date:           "2026-04-27",
		MarketSnapshot: "上证指数 3094.67（+1.23%），深证成指 10001.00（-0.45%），创业板指 2000.10（+0.67%）",
	})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}
	if !strings.Contains(detail.Content, "市场概览：上证指数 3094.67") {
		t.Fatalf("rendered content should include market snapshot, got %s", detail.Content)
	}
	if strings.Contains(detail.Content, "{{marketSnapshot}}") {
		t.Fatalf("marketSnapshot variable should be rendered, got %s", detail.Content)
	}
}

func TestCreateDailyReviewRejectsInvalidDate(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	_, err = service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "20260427"})
	if err == nil || !strings.Contains(err.Error(), "复盘日期格式错误") {
		t.Fatalf("expected date format error, got %v", err)
	}
}

func TestCreateDailyReviewRejectsDuplicateDateWithoutOverwrite(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"}); err != nil {
		t.Fatalf("first CreateDailyReview returned error: %v", err)
	}
	target := filepath.Join(dataDir, "articles", "2026-04-27.md")
	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatalf("write original marker: %v", err)
	}

	_, err = service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err == nil || !strings.Contains(err.Error(), "该日期复盘已存在") {
		t.Fatalf("expected duplicate date error, got %v", err)
	}

	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if string(content) != "original" {
		t.Fatalf("duplicate create should not overwrite file, got %q", string(content))
	}
}

func TestGetArticleAndListArticlesOrdering(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-26"}); err != nil {
		t.Fatalf("create first article: %v", err)
	}
	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"}); err != nil {
		t.Fatalf("create second article: %v", err)
	}

	detail, err := service.GetArticle("daily-2026-04-27")
	if err != nil {
		t.Fatalf("GetArticle returned error: %v", err)
	}
	if detail.Article.Date != "2026-04-27" || !strings.Contains(detail.Content, "2026-04-27") {
		t.Fatalf("unexpected article detail: %+v", detail)
	}

	result, err := service.ListArticles(models.ReviewListRequest{})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if result.Total != 3 || len(result.Items) != 3 {
		t.Fatalf("expected summary plus two daily reviews, got total=%d len=%d", result.Total, len(result.Items))
	}
	if result.Items[0].Type != models.ReviewTypeSummary {
		t.Fatalf("summary should be pinned first, got %+v", result.Items)
	}
	if result.Items[1].Date != "2026-04-27" || result.Items[2].Date != "2026-04-26" {
		t.Fatalf("daily reviews should sort by date desc, got %+v", result.Items)
	}
}

func TestListArticlesSearchDateAndEmptyResult(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-26", Title: "普通复盘"}); err != nil {
		t.Fatalf("create first article: %v", err)
	}
	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27", Title: "纪律问题复盘"}); err != nil {
		t.Fatalf("create second article: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "articles", "2026-04-27.md"), []byte("纪律问题正文"), 0644); err != nil {
		t.Fatalf("overwrite article body: %v", err)
	}

	result, err := service.ListArticles(models.ReviewListRequest{
		Query:     "纪律问题",
		Type:      models.ReviewTypeDaily,
		StartDate: "2026-04-27",
		EndDate:   "2026-04-27",
	})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if result.Total != 1 || result.Items[0].Date != "2026-04-27" {
		t.Fatalf("expected one filtered article, got %+v", result)
	}

	empty, err := service.ListArticles(models.ReviewListRequest{Query: "不存在关键词"})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if empty.Total != 0 || len(empty.Items) != 0 {
		t.Fatalf("expected empty result, got %+v", empty)
	}
}

func TestListArticlesPaginationLimit(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	for day := 0; day < 205; day++ {
		date := start.AddDate(0, 0, day).Format("2006-01-02")
		if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: date}); err != nil {
			t.Fatalf("create article %s: %v", date, err)
		}
	}

	result, err := service.ListArticles(models.ReviewListRequest{PageSize: 1000})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if len(result.Items) != 200 {
		t.Fatalf("page size should be capped at 200, got %d", len(result.Items))
	}
	if result.Total != 206 {
		t.Fatalf("total should include all rows before paging, got %d", result.Total)
	}
}

func TestSaveArticleUpdatesContentAndIndex(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	created, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}

	saved, err := service.SaveArticle(models.SaveReviewArticleRequest{
		ID:      created.Article.ID,
		Title:   "",
		Content: "# 保存后标题\n\n![a](../pictures/2026/04/27/a.png)\n![b](pictures/2026/04/27/b.png)\n\n纪律问题正文",
	})
	if err != nil {
		t.Fatalf("SaveArticle returned error: %v", err)
	}
	if saved.Article.Title != "保存后标题" {
		t.Fatalf("title should be inferred from H1, got %q", saved.Article.Title)
	}
	if saved.Article.ImageCount != 2 {
		t.Fatalf("image count should be synced, got %d", saved.Article.ImageCount)
	}
	if !strings.Contains(saved.Content, "纪律问题正文") || !strings.Contains(saved.Content, "type: daily_review") {
		t.Fatalf("saved content should include body and legal front matter, got %s", saved.Content)
	}

	detail, err := service.GetArticle(created.Article.ID)
	if err != nil {
		t.Fatalf("GetArticle returned error: %v", err)
	}
	if detail.Article.ImageCount != 2 || !strings.Contains(detail.Content, "纪律问题正文") {
		t.Fatalf("detail should reflect saved content and image count: %+v", detail)
	}
}

func TestSaveArticleAllowsEmptyContent(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	created, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}

	saved, err := service.SaveArticle(models.SaveReviewArticleRequest{
		ID:      created.Article.ID,
		Content: "",
	})
	if err != nil {
		t.Fatalf("SaveArticle returned error: %v", err)
	}
	if !strings.Contains(saved.Content, "type: daily_review") {
		t.Fatalf("empty save should still write front matter, got %s", saved.Content)
	}
}

func TestSaveArticleRewritesBrokenFrontMatter(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	created, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}

	saved, err := service.SaveArticle(models.SaveReviewArticleRequest{
		ID: created.Article.ID,
		Content: `---
type: daily_review
title: 坏掉的头

# 修复后标题

正文仍然保留`,
	})
	if err != nil {
		t.Fatalf("SaveArticle returned error: %v", err)
	}
	if !strings.Contains(saved.Content, "---\n\n---") && strings.Count(saved.Content, "---") < 2 {
		t.Fatalf("saved content should contain rewritten front matter, got %s", saved.Content)
	}
	if !strings.Contains(saved.Content, "正文仍然保留") {
		t.Fatalf("body should be preserved, got %s", saved.Content)
	}
}

func TestDeleteArticleRemovesDailyReviewButKeepsPicture(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	created, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if err != nil {
		t.Fatalf("CreateDailyReview returned error: %v", err)
	}
	picturePath := filepath.Join(dataDir, "pictures", "2026", "04", "27", "a.png")
	if err := os.MkdirAll(filepath.Dir(picturePath), 0755); err != nil {
		t.Fatalf("mkdir picture dir: %v", err)
	}
	if err := os.WriteFile(picturePath, []byte("png"), 0644); err != nil {
		t.Fatalf("write picture: %v", err)
	}
	if _, err := service.db.Exec(`
		INSERT INTO review_assets (id, article_id, source, file_path, mime_type, size, created_at, updated_at)
		VALUES ('asset-1', ?, 'local', 'pictures/2026/04/27/a.png', 'image/png', 3, 1, 1)
	`, created.Article.ID); err != nil {
		t.Fatalf("insert asset: %v", err)
	}

	if err := service.DeleteArticle(created.Article.ID); err != nil {
		t.Fatalf("DeleteArticle returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "articles", "2026-04-27.md")); !os.IsNotExist(err) {
		t.Fatalf("article file should be deleted, stat err=%v", err)
	}
	if _, err := os.Stat(picturePath); err != nil {
		t.Fatalf("picture should be kept: %v", err)
	}

	result, err := service.ListArticles(models.ReviewListRequest{Type: models.ReviewTypeDaily})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("deleted daily review should not appear in list: %+v", result)
	}

	var articleID string
	if err := service.db.QueryRow(`SELECT article_id FROM review_assets WHERE id = 'asset-1'`).Scan(&articleID); err != nil {
		t.Fatalf("query asset: %v", err)
	}
	if articleID != "" {
		t.Fatalf("asset article_id should be cleared, got %q", articleID)
	}
}

func TestDeleteArticleRejectsSummaryAndMissingArticle(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if err := service.DeleteArticle("summary"); err == nil || !strings.Contains(err.Error(), "总复盘总结不可删除") {
		t.Fatalf("expected summary delete error, got %v", err)
	}
	if err := service.DeleteArticle("missing"); err == nil || !strings.Contains(err.Error(), "文章不存在") {
		t.Fatalf("expected missing article error, got %v", err)
	}
}

func TestRebuildIndexRestoresArticlesFromMarkdownFiles(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-26"}); err != nil {
		t.Fatalf("create first article: %v", err)
	}
	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"}); err != nil {
		t.Fatalf("create second article: %v", err)
	}
	if _, err := service.db.Exec(`DELETE FROM review_articles`); err != nil {
		t.Fatalf("clear articles: %v", err)
	}

	if err := service.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex returned error: %v", err)
	}

	result, err := service.ListArticles(models.ReviewListRequest{})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if result.Total != 3 {
		t.Fatalf("expected summary and two daily reviews after rebuild, got %+v", result)
	}
}

func TestRebuildIndexSyncsExternalChangesAndSkipsInvalidFiles(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if _, err := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"}); err != nil {
		t.Fatalf("create article: %v", err)
	}
	changed := `---
type: daily_review
date: 2026-04-27
title: 外部修改标题
tags: [纪律]
stocks: [600000]
profitLoss: -2.5
emotion: anxious
disciplineScore: 6
---

# 外部修改标题

外部修改后的正文`
	if err := os.WriteFile(filepath.Join(dataDir, "articles", "2026-04-27.md"), []byte(changed), 0644); err != nil {
		t.Fatalf("write changed article: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "articles", "abc.md"), []byte("# should skip"), 0644); err != nil {
		t.Fatalf("write invalid article: %v", err)
	}

	if err := service.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex returned error: %v", err)
	}

	result, err := service.ListArticles(models.ReviewListRequest{Tags: []string{"纪律"}})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected one tag-filtered result, got %+v", result)
	}
	article := result.Items[0]
	if article.Title != "外部修改标题" || article.Stocks[0] != "600000" || article.DisciplineScore == nil || *article.DisciplineScore != 6 {
		t.Fatalf("external metadata should be synced, got %+v", article)
	}

	all, err := service.ListArticles(models.ReviewListRequest{})
	if err != nil {
		t.Fatalf("ListArticles returned error: %v", err)
	}
	for _, item := range all.Items {
		if item.FilePath == "articles/abc.md" {
			t.Fatalf("invalid markdown filename should be skipped")
		}
	}
}
