package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var allowedReviewPathRoots = map[string]struct{}{
	"articles":         {},
	"pictures":         {},
	"review_templates": {},
}

func ResolveReviewSafePath(dataDir, relativePath string) (string, string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", "", fmt.Errorf("复盘路径不能为空")
	}
	if strings.Contains(relativePath, "\x00") {
		return "", "", fmt.Errorf("复盘路径非法: 包含空字节")
	}
	if strings.Contains(relativePath, `\`) {
		return "", "", fmt.Errorf("复盘路径非法: 请使用 / 分隔")
	}
	if filepath.IsAbs(relativePath) || filepath.VolumeName(relativePath) != "" || strings.HasPrefix(relativePath, `\\`) {
		return "", "", fmt.Errorf("复盘路径非法: 不允许绝对路径")
	}

	cleanSlash := filepath.ToSlash(filepath.Clean(filepath.FromSlash(relativePath)))
	if cleanSlash == "." || strings.HasPrefix(cleanSlash, "../") || cleanSlash == ".." {
		return "", "", fmt.Errorf("复盘路径非法: 不允许路径穿越")
	}

	parts := strings.Split(cleanSlash, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("复盘路径非法: 缺少文件名")
	}
	if _, ok := allowedReviewPathRoots[parts[0]]; !ok {
		return "", "", fmt.Errorf("复盘路径非法: 只能位于 articles、pictures 或 review_templates 目录")
	}

	base, err := filepath.Abs(dataDir)
	if err != nil {
		return "", "", fmt.Errorf("解析复盘数据目录失败: %w", err)
	}
	target, err := filepath.Abs(filepath.Join(base, filepath.FromSlash(cleanSlash)))
	if err != nil {
		return "", "", fmt.Errorf("解析复盘路径失败: %w", err)
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", "", fmt.Errorf("校验复盘路径失败: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("复盘路径非法: 不允许逃逸数据目录")
	}

	return target, cleanSlash, nil
}

func AtomicWriteReviewFile(target string, data []byte) error {
	return atomicWriteReviewFileWithTempDir(target, data, filepath.Dir(target))
}

func atomicWriteReviewFileWithTempDir(target string, data []byte, tempDir string) error {
	tmp, err := os.CreateTemp(tempDir, "."+filepath.Base(target)+".*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时复盘文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时复盘文件失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("刷新临时复盘文件失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时复盘文件失败: %w", err)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		if removeErr := os.Remove(target); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("替换复盘文件失败: %w", err)
		}
		if renameErr := os.Rename(tmpPath, target); renameErr != nil {
			return fmt.Errorf("替换复盘文件失败: %w", renameErr)
		}
	}
	committed = true
	return nil
}
