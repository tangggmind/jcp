package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetDataDirUsesExecutableSiblingDataDirectory(t *testing.T) {
	exeDir := t.TempDir()
	legacyDir := t.TempDir()
	restore := overridePathSources(t, exeDir, legacyDir)
	defer restore()

	dataDir := GetDataDir()
	expected := filepath.Join(exeDir, "data")
	if dataDir != expected {
		t.Fatalf("expected data dir %q, got %q", expected, dataDir)
	}
	if info, err := os.Stat(expected); err != nil || !info.IsDir() {
		t.Fatalf("expected data dir to be created: %v", err)
	}
}

func TestGetDataDirMigratesLegacyDataIntoEmptyDataDirectory(t *testing.T) {
	exeDir := t.TempDir()
	legacyDir := t.TempDir()
	restore := overridePathSources(t, exeDir, legacyDir)
	defer restore()

	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte(`{"theme":"legacy"}`), 0644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(legacyDir, "sessions"), 0755); err != nil {
		t.Fatalf("create legacy sessions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "sessions", "sh600000.json"), []byte(`{"stockCode":"sh600000"}`), 0644); err != nil {
		t.Fatalf("write legacy session: %v", err)
	}

	dataDir := GetDataDir()
	if _, err := os.Stat(filepath.Join(dataDir, "config.json")); err != nil {
		t.Fatalf("expected legacy config to be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "sessions", "sh600000.json")); err != nil {
		t.Fatalf("expected nested legacy data to be copied: %v", err)
	}
}

func TestGetDataDirDoesNotOverwriteExistingDataDirectory(t *testing.T) {
	exeDir := t.TempDir()
	legacyDir := t.TempDir()
	restore := overridePathSources(t, exeDir, legacyDir)
	defer restore()

	dataDir := filepath.Join(exeDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("create data dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(`{"theme":"current"}`), 0644); err != nil {
		t.Fatalf("write current config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "config.json"), []byte(`{"theme":"legacy"}`), 0644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	resolved := GetDataDir()
	content, err := os.ReadFile(filepath.Join(resolved, "config.json"))
	if err != nil {
		t.Fatalf("read current config: %v", err)
	}
	if string(content) != `{"theme":"current"}` {
		t.Fatalf("existing data should not be overwritten, got %s", content)
	}
}

func TestEnsureCacheDirCreatesSubDirectoryUnderExecutableData(t *testing.T) {
	exeDir := t.TempDir()
	legacyDir := t.TempDir()
	restore := overridePathSources(t, exeDir, legacyDir)
	defer restore()

	cacheDir := EnsureCacheDir(filepath.Join("market", "tdx"))
	expected := filepath.Join(exeDir, "data", "cache", "market", "tdx")
	if cacheDir != expected {
		t.Fatalf("expected cache dir %q, got %q", expected, cacheDir)
	}
	if info, err := os.Stat(expected); err != nil || !info.IsDir() {
		t.Fatalf("expected cache dir to be created: %v", err)
	}
}

func overridePathSources(t *testing.T, exeDir, legacyDir string) func() {
	t.Helper()
	previousExecutableDir := executableDir
	previousLegacyDataDir := legacyDataDir
	executableDir = func() string { return exeDir }
	legacyDataDir = func() string { return legacyDir }
	return func() {
		executableDir = previousExecutableDir
		legacyDataDir = previousLegacyDataDir
	}
}
