package docu

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	GOROOT  = strOr(os.Getenv("GOROOT"), runtime.GOROOT())
	GOPATHS = filepath.SplitList(os.Getenv("GOPATH"))
)

func strOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Abs 返回 path 的绝对路径.
// 如果 path 疑似绝对路径返回 path.
// 否则在 GOROOT, GOPATHS 中搜索 path 并返回绝对路径.
// 如果未找到返回 path.
func Abs(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if path[0] == '.' {
		if abs, err := filepath.Abs(path); err == nil && exists(abs) {
			return abs
		}
	}

	abs, err := filepath.Abs(filepath.Join(GOROOT, "src", path))
	if err == nil && exists(abs) {
		return abs
	}

	for _, gopath := range GOPATHS {
		abs, err := filepath.Abs(filepath.Join(gopath, "src", path))
		if err == nil && exists(abs) {
			return abs
		}
	}

	return path
}
