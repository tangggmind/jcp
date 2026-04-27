package main

import (
	"strings"
	"testing"

	"github.com/run-bigpig/jcp/internal/models"
	"github.com/run-bigpig/jcp/internal/services"
)

func TestReviewAPIReturnsErrorWhenServiceMissing(t *testing.T) {
	app := &App{}

	list := app.GetReviewArticles(models.ReviewListRequest{})
	if list.Error == "" {
		t.Fatalf("expected list error when review service is nil")
	}

	detail := app.GetReviewArticle("missing")
	if detail.Error == "" {
		t.Fatalf("expected detail error when review service is nil")
	}

	if got := app.DeleteReviewArticle("missing"); !strings.Contains(got, "复盘服务未初始化") {
		t.Fatalf("expected delete error, got %q", got)
	}
}

func TestReviewAPICreateReadAndRebuild(t *testing.T) {
	service, err := services.NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	app := &App{reviewService: service}

	created := app.CreateDailyReview(models.CreateDailyReviewRequest{Date: "2026-04-27"})
	if created.Error != "" {
		t.Fatalf("CreateDailyReview returned error: %s", created.Error)
	}

	got := app.GetReviewArticle(created.Article.ID)
	if got.Error != "" || got.Article.ID != created.Article.ID {
		t.Fatalf("GetReviewArticle returned unexpected detail: %+v", got)
	}

	if result := app.RebuildReviewIndex(); result != "success" {
		t.Fatalf("RebuildReviewIndex should return success, got %q", result)
	}
}

func TestFormatReviewMarketSnapshot(t *testing.T) {
	got := formatReviewMarketSnapshot([]models.MarketIndex{
		{Code: "sz399006", Price: 2000.1, ChangePercent: -0.67},
		{Code: "sh000001", Price: 3094.668, ChangePercent: 1.23},
		{Code: "sz399001", Price: 10001, ChangePercent: 0.45},
	})
	expected := "上证指数 3094.67（+1.23%），深证成指 10001.00（+0.45%），创业板指 2000.10（-0.67%）"
	if got != expected {
		t.Fatalf("unexpected market snapshot:\nwant %s\n got %s", expected, got)
	}
}
