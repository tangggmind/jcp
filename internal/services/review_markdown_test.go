package services

import (
	"strings"
	"testing"

	"github.com/run-bigpig/jcp/internal/models"
)

func TestParseReviewMarkdownFrontMatter(t *testing.T) {
	raw := `---
type: daily_review
date: 2026-04-27
title: 2026-04-27 每日复盘
templateId: default-daily
templateName: 标准每日复盘
summary: 今日严格执行计划
tags: [纪律, 回撤]
stocks: [600000, 000001]
profitLoss: 12.5
emotion: calm
disciplineScore: 8
createdAt: 2026-04-27T15:30:00+08:00
updatedAt: 2026-04-27T16:30:00+08:00
---

# 2026-04-27 每日复盘

正文内容`

	parsed := ParseReviewMarkdown(raw)
	if parsed.Warning != "" {
		t.Fatalf("unexpected warning: %s", parsed.Warning)
	}
	if !parsed.HasFrontMatter || parsed.NeedsRewrite {
		t.Fatalf("expected valid front matter: %+v", parsed)
	}
	if parsed.Meta.Type != models.ReviewTypeDaily || parsed.Meta.Date != "2026-04-27" {
		t.Fatalf("unexpected type/date: %+v", parsed.Meta)
	}
	if parsed.Meta.TemplateID != "default-daily" || parsed.Meta.TemplateName != "标准每日复盘" {
		t.Fatalf("unexpected template metadata: %+v", parsed.Meta)
	}
	if parsed.Meta.Summary != "今日严格执行计划" || parsed.Summary != "今日严格执行计划" {
		t.Fatalf("unexpected summary: meta=%q result=%q", parsed.Meta.Summary, parsed.Summary)
	}
	if len(parsed.Meta.Tags) != 2 || parsed.Meta.Tags[0] != "纪律" {
		t.Fatalf("unexpected tags: %+v", parsed.Meta.Tags)
	}
	if parsed.Meta.DisciplineScore == nil || *parsed.Meta.DisciplineScore != 8 {
		t.Fatalf("unexpected discipline score: %+v", parsed.Meta.DisciplineScore)
	}
	if !strings.Contains(parsed.Content, "正文内容") {
		t.Fatalf("expected body content, got %q", parsed.Content)
	}
}

func TestParseReviewMarkdownSummaryFallback(t *testing.T) {
	raw := `# 标题

## 今日操作

- **买入** [平安银行](https://example.invalid) ，按计划执行。
- 继续观察回撤。`

	parsed := ParseReviewMarkdown(raw)
	if parsed.HasFrontMatter {
		t.Fatalf("expected markdown without front matter")
	}
	if strings.Contains(parsed.Summary, "#") || strings.Contains(parsed.Summary, "**") || strings.Contains(parsed.Summary, "](") {
		t.Fatalf("summary should remove markdown markers, got %q", parsed.Summary)
	}
	if len([]rune(parsed.Summary)) > 160 {
		t.Fatalf("summary should be truncated to 160 runes, got %d", len([]rune(parsed.Summary)))
	}
	if !strings.Contains(parsed.Summary, "买入") {
		t.Fatalf("summary should include body text, got %q", parsed.Summary)
	}
}

func TestParseReviewMarkdownBrokenFrontMatter(t *testing.T) {
	raw := `---
type: daily_review
title: 未闭合 Front Matter

# 正文标题

正文保留`

	parsed := ParseReviewMarkdown(raw)
	if !parsed.NeedsRewrite {
		t.Fatalf("broken front matter should request rewrite")
	}
	if parsed.Warning == "" {
		t.Fatalf("broken front matter should include warning")
	}
	if !strings.Contains(parsed.Content, "正文保留") {
		t.Fatalf("body should still be readable, got %q", parsed.Content)
	}
}

func TestScanLocalReviewImageRefs(t *testing.T) {
	content := `![remote](https://example.invalid/a.png)
![local](../pictures/2026/04/27/a.png)
![data](data:image/png;base64,abc)
![plain](pictures/2026/04/27/b.png)
![absolute](/tmp/c.png)`

	refs := ScanLocalReviewImageRefs(content)
	if len(refs) != 2 {
		t.Fatalf("expected two local image refs, got %+v", refs)
	}
	if refs[0] != "../pictures/2026/04/27/a.png" || refs[1] != "pictures/2026/04/27/b.png" {
		t.Fatalf("unexpected local refs: %+v", refs)
	}
}
