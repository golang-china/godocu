package docu

import (
	"go/ast"
	"path/filepath"
	"strings"
)

// DefaultFilter 缺省的过滤规则. 过滤掉测试文件
func DefaultFilter(name string) bool {
	ext := filepath.Ext(name)
	if ext == ".go" {
		return !strings.HasSuffix(name, "_test.go")
	}
	return false
}

// ExportedFilter 剔除非导出声明
func ExportedFilter(file *ast.File) {

}
