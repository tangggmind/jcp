package services

import (
	"strings"
	"testing"

	"github.com/run-bigpig/jcp/internal/models"
)

func TestCompareReviewArticlesByIDsAndRange(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()
	a1, _ := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-26"})
	a2, _ := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})

	result, err := service.Compare(models.CompareReviewRequest{ArticleIDs: []string{a1.Article.ID, "summary", a2.Article.ID}})
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if len(result.Items) != 2 || result.Items[0].Date != "2026-04-26" || result.Items[1].Date != "2026-04-27" {
		t.Fatalf("unexpected compare result: %+v", result.Items)
	}

	byRange, err := service.Compare(models.CompareReviewRequest{StartDate: "2026-04-26", EndDate: "2026-04-27"})
	if err != nil {
		t.Fatalf("Compare by range returned error: %v", err)
	}
	if len(byRange.Items) != 2 {
		t.Fatalf("expected two range items, got %+v", byRange)
	}
}

func TestCompareReviewArticlesRequiresTwoDailyReviews(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()
	article, _ := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})

	_, err = service.Compare(models.CompareReviewRequest{ArticleIDs: []string{article.Article.ID}})
	if err == nil || !strings.Contains(err.Error(), "至少选择 2 篇") {
		t.Fatalf("expected min selection error, got %v", err)
	}
}

func TestCompareReviewArticlesExtractsSectionsAndStats(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()
	a1, _ := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-26"})
	a2, _ := service.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if _, err := service.SaveArticle(models.SaveReviewArticleRequest{ID: a1.Article.ID, Content: "---\ntags: [纪律]\nstocks: [600000]\n---\n# A\n\n## 今日市场\n\n市场段落\n"}); err != nil {
		t.Fatalf("save first: %v", err)
	}
	if _, err := service.SaveArticle(models.SaveReviewArticleRequest{ID: a2.Article.ID, Content: "---\ntags: [纪律, 计划]\nstocks: [600000, 000001]\n---\n# B\n\n## 今日市场\n\n" + strings.Repeat("长", 400)}); err != nil {
		t.Fatalf("save second: %v", err)
	}

	result, err := service.Compare(models.CompareReviewRequest{ArticleIDs: []string{a1.Article.ID, a2.Article.ID}})
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if result.Items[0].Sections["今日市场"] != "市场段落" {
		t.Fatalf("expected section content, got %+v", result.Items[0].Sections)
	}
	if len([]rune(result.Items[1].Sections["今日市场"])) != 300 {
		t.Fatalf("long section should be truncated to 300 runes")
	}
	if result.TagStats[0].Name != "纪律" || result.TagStats[0].Count != 2 {
		t.Fatalf("unexpected tag stats: %+v", result.TagStats)
	}
	if result.StockStats[0].Name != "600000" || result.StockStats[0].Count != 2 {
		t.Fatalf("unexpected stock stats: %+v", result.StockStats)
	}
}
