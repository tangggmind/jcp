package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveReviewSafePathAllowsKnownRoots(t *testing.T) {
	dataDir := t.TempDir()

	resolved, markdownPath, err := ResolveReviewSafePath(dataDir, "articles/2026-04-27.md")
	if err != nil {
		t.Fatalf("ResolveReviewSafePath returned error: %v", err)
	}
	if !strings.HasPrefix(resolved, filepath.Clean(dataDir)+string(os.PathSeparator)) {
		t.Fatalf("resolved path should stay in dataDir, got %s", resolved)
	}
	if markdownPath != "articles/2026-04-27.md" {
		t.Fatalf("markdown path should use slash separators, got %s", markdownPath)
	}
}

func TestResolveReviewSafePathRejectsTraversalAndAbsolutePaths(t *testing.T) {
	dataDir := t.TempDir()

	for _, input := range []string{
		"../secret.txt",
		"articles/../../secret.txt",
		filepath.Join(dataDir, "articles", "x.md"),
		`C:\temp\secret.txt`,
		`\\server\share\secret.txt`,
		"articles/bad\x00name.md",
		"sessions/2026-04-27.md",
	} {
		if _, _, err := ResolveReviewSafePath(dataDir, input); err == nil {
			t.Fatalf("expected invalid path error for %q", input)
		}
	}
}

func TestAtomicWriteReviewFileReplacesContent(t *testing.T) {
	dataDir := t.TempDir()
	target := filepath.Join(dataDir, "articles", "2026-04-27.md")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatalf("write old content: %v", err)
	}

	if err := AtomicWriteReviewFile(target, []byte("new")); err != nil {
		t.Fatalf("AtomicWriteReviewFile returned error: %v", err)
	}

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(content) != "new" {
		t.Fatalf("expected replaced content, got %q", string(content))
	}
}

func TestAtomicWriteReviewFileKeepsOldContentWhenTempWriteFails(t *testing.T) {
	dataDir := t.TempDir()
	target := filepath.Join(dataDir, "articles", "2026-04-27.md")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatalf("write old content: %v", err)
	}

	err := atomicWriteReviewFileWithTempDir(target, []byte("new"), filepath.Join(dataDir, "missing-temp-dir"))
	if err == nil {
		t.Fatalf("expected temp write error")
	}

	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if string(content) != "old" {
		t.Fatalf("old content should remain after failed write, got %q", string(content))
	}
}
