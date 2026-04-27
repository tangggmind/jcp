package paths

import (
	"io"
	"os"
	"path/filepath"
)

var executableDir = defaultExecutableDir
var legacyDataDir = defaultLegacyDataDir

// GetDataDir 获取应用数据目录
func GetDataDir() string {
	dataDir := filepath.Join(executableDir(), "data")
	_ = os.MkdirAll(dataDir, 0755)
	migrateLegacyDataDir(dataDir)
	return dataDir
}

func defaultExecutableDir() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		if wd, wdErr := os.Getwd(); wdErr == nil && wd != "" {
			return wd
		}
		return "."
	}
	return filepath.Dir(exe)
}

func defaultLegacyDataDir() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil || userConfigDir == "" {
		return ""
	}
	return filepath.Join(userConfigDir, "jcp")
}

func migrateLegacyDataDir(dataDir string) {
	legacyDir := legacyDataDir()
	if legacyDir == "" || samePath(legacyDir, dataDir) {
		return
	}
	if !dirExists(legacyDir) || !dirIsEmpty(dataDir) {
		return
	}
	_ = copyDir(legacyDir, dataDir)
}

func dirExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

func dirIsEmpty(dir string) bool {
	entries, err := os.ReadDir(dir)
	return err == nil && len(entries) == 0
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relativePath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, relativePath)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// GetCacheDir 获取缓存目录
func GetCacheDir() string {
	return filepath.Join(GetDataDir(), "cache")
}

// EnsureCacheDir 确保缓存目录存在并返回路径
func EnsureCacheDir(subDir string) string {
	dir := filepath.Join(GetCacheDir(), subDir)
	os.MkdirAll(dir, 0755)
	return dir
}
