package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/run-bigpig/jcp/internal/models"
)

func TestSaveTemplateCreatesCustomTemplateAndDefaultSwitch(t *testing.T) {
	dataDir := t.TempDir()
	service, err := NewReviewService(dataDir)
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	template, err := service.SaveTemplate(models.SaveReviewTemplateRequest{
		Name:        "我的模板",
		Description: "自定义",
		Content:     "# {{title}}\n\n## 交易计划",
		IsDefault:   true,
	})
	if err != nil {
		t.Fatalf("SaveTemplate returned error: %v", err)
	}
	if template.ID == "" || template.IsBuiltin || !template.IsDefault {
		t.Fatalf("unexpected template: %+v", template)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "review_templates", template.ID+".md")); err != nil {
		t.Fatalf("expected template file: %v", err)
	}

	templates, err := service.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates returned error: %v", err)
	}
	defaultCount := 0
	for _, item := range templates {
		if item.IsDefault {
			defaultCount++
		}
	}
	if defaultCount != 1 {
		t.Fatalf("expected one default template, got %d", defaultCount)
	}
}

func TestDeleteTemplateRules(t *testing.T) {
	service, err := NewReviewService(t.TempDir())
	if err != nil {
		t.Fatalf("NewReviewService returned error: %v", err)
	}
	defer service.Close()

	if err := service.DeleteTemplate("default-daily"); err == nil || !strings.Contains(err.Error(), "内置模板不可删除") {
		t.Fatalf("expected builtin delete error, got %v", err)
	}
	if _, err := service.SaveTemplate(models.SaveReviewTemplateRequest{Name: "", Content: "# x"}); err == nil || !strings.Contains(err.Error(), "模板名称必填") {
		t.Fatalf("expected name required error, got %v", err)
	}
	custom, err := service.SaveTemplate(models.SaveReviewTemplateRequest{Name: "可删除模板", Content: "# x"})
	if err != nil {
		t.Fatalf("SaveTemplate returned error: %v", err)
	}
	if err := service.DeleteTemplate(custom.ID); err != nil {
		t.Fatalf("DeleteTemplate returned error: %v", err)
	}
}
