package models

import (
	"encoding/json"
	"testing"
)

func TestReviewArticleJSONContract(t *testing.T) {
	score := 8
	article := ReviewArticle{
		ID:              "review-2026-04-27",
		Type:            ReviewTypeDaily,
		Date:            "2026-04-27",
		Title:           "2026-04-27 每日复盘",
		FilePath:        "articles/2026-04-27.md",
		TemplateID:      "default-daily",
		TemplateName:    "默认每日复盘模板",
		Summary:         "测试摘要",
		Tags:            []string{},
		Stocks:          []string{},
		ProfitLoss:      12.5,
		Emotion:         "calm",
		DisciplineScore: &score,
		ImageCount:      2,
		CreatedAt:       1714180800,
		UpdatedAt:       1714180900,
	}

	raw, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("marshal ReviewArticle: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal ReviewArticle: %v", err)
	}

	for _, key := range []string{
		"id", "type", "date", "title", "filePath", "templateId", "templateName",
		"summary", "tags", "stocks", "profitLoss", "emotion", "disciplineScore",
		"imageCount", "createdAt", "updatedAt",
	} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing json field %q in %s", key, string(raw))
		}
	}

	if ReviewTypeDaily != "daily_review" || ReviewTypeSummary != "summary_review" {
		t.Fatalf("unexpected review type constants")
	}
	if ReviewSourcePaste != "paste" || ReviewSourceDownload != "download" || ReviewSourceLocal != "local" {
		t.Fatalf("unexpected review source constants")
	}
}

func TestReviewArticleEmptyCollectionsAndNilScore(t *testing.T) {
	article := ReviewArticle{
		ID:     "empty",
		Tags:   []string{},
		Stocks: []string{},
	}

	raw, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("marshal ReviewArticle: %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal ReviewArticle: %v", err)
	}

	if string(got["tags"]) != "[]" {
		t.Fatalf("tags should serialize as empty array, got %s", got["tags"])
	}
	if string(got["stocks"]) != "[]" {
		t.Fatalf("stocks should serialize as empty array, got %s", got["stocks"])
	}
	if _, ok := got["disciplineScore"]; ok {
		t.Fatalf("nil disciplineScore should be omitted, got %s", got["disciplineScore"])
	}
}

func TestCompareReviewResultJSONContract(t *testing.T) {
	score := 9
	result := CompareReviewResult{
		Items: []CompareReviewItem{
			{
				ArticleID:       "a1",
				Date:            "2026-04-27",
				Title:           "复盘",
				Summary:         "摘要",
				Tags:            []string{"纪律"},
				Stocks:          []string{"600000"},
				ProfitLoss:      -1.5,
				Emotion:         "anxious",
				DisciplineScore: &score,
				Sections:        map[string]string{"今日操作": "减仓"},
			},
		},
		TagStats:   []CompareStatItem{{Name: "纪律", Count: 1}},
		StockStats: []CompareStatItem{{Name: "600000", Count: 1}},
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal CompareReviewResult: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal CompareReviewResult: %v", err)
	}
	for _, key := range []string{"items", "tagStats", "stockStats"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("missing json field %q in %s", key, string(raw))
		}
	}
}
