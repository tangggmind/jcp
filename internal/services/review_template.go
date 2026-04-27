package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/run-bigpig/jcp/internal/models"
)

var reviewTemplateIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func (s *ReviewService) SaveTemplate(req models.SaveReviewTemplateRequest) (models.ReviewTemplate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return models.ReviewTemplate{}, fmt.Errorf("模板名称必填")
	}
	id := strings.TrimSpace(req.ID)
	if id == "" {
		id = "tpl-" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	}
	if !reviewTemplateIDPattern.MatchString(id) {
		return models.ReviewTemplate{}, fmt.Errorf("模板 ID 非法")
	}

	existing, err := s.getTemplateByIDNoLock(id)
	isBuiltin := false
	createdAt := time.Now().UnixMilli()
	if err == nil {
		isBuiltin = existing.IsBuiltin
		createdAt = existing.CreatedAt
	}
	relativePath := filepath.ToSlash(filepath.Join("review_templates", id+".md"))
	target, filePath, err := ResolveReviewSafePath(s.dataDir, relativePath)
	if err != nil {
		return models.ReviewTemplate{}, err
	}
	if err := AtomicWriteReviewFile(target, []byte(req.Content)); err != nil {
		return models.ReviewTemplate{}, err
	}

	now := time.Now().UnixMilli()
	tx, err := s.db.Begin()
	if err != nil {
		return models.ReviewTemplate{}, fmt.Errorf("保存复盘模板失败: %w", err)
	}
	defer tx.Rollback()
	if req.IsDefault {
		if _, err := tx.Exec(`UPDATE review_templates SET is_default = 0`); err != nil {
			return models.ReviewTemplate{}, fmt.Errorf("保存复盘模板失败: %w", err)
		}
	}
	if _, err := tx.Exec(`
		INSERT INTO review_templates (id, name, description, file_path, is_builtin, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			file_path = excluded.file_path,
			is_builtin = excluded.is_builtin,
			is_default = excluded.is_default,
			updated_at = excluded.updated_at
	`, id, name, req.Description, filePath, boolToInt(isBuiltin), boolToInt(req.IsDefault), createdAt, now); err != nil {
		return models.ReviewTemplate{}, fmt.Errorf("保存复盘模板失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return models.ReviewTemplate{}, fmt.Errorf("保存复盘模板失败: %w", err)
	}

	return s.getTemplateByIDNoLock(id)
}

func (s *ReviewService) DeleteTemplate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	template, err := s.getTemplateByIDNoLock(id)
	if err != nil {
		return err
	}
	if template.IsBuiltin {
		return fmt.Errorf("内置模板不可删除")
	}
	if template.IsDefault {
		return fmt.Errorf("默认模板不可删除，请先设置其他模板为默认")
	}
	target, _, err := ResolveReviewSafePath(s.dataDir, filepath.ToSlash(filepath.Join("review_templates", template.ID+".md")))
	if err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除复盘模板文件失败: %w", err)
	}
	if _, err := s.db.Exec(`DELETE FROM review_templates WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除复盘模板失败: %w", err)
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
