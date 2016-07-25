// +build go1.5

// coming soon....
package docu

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/tools/godoc/vfs"
)

// Docu 复合 token.FileSet, ast.Package 提供 Go doc 支持.
type Docu struct {
	*token.FileSet
	// Astpkg 以 import paths 做 map key
	Astpkg map[string]*ast.Package
	// Filter 用于生成 Astpkg 时过滤目录名和文件名 name.
	// name 不包括上级路径.
	Filter func(name string) bool
}

// New 返回使用 DefaultFilter 进行过滤的 Docu.
func New() Docu {
	return Docu{token.NewFileSet(), make(map[string]*ast.Package), DefaultFilter}
}

// Parse 返回解析 Go 文件, 包名称或包目录发生的错误.
// source 可以是 nil, []byte, string, io.Reader 或 vfs.FileSystem.
func (du *Docu) Parse(path string, source interface{}) (err error) {
	var info []os.FileInfo
	var fs vfs.FileSystem
	var ok bool
	if source == nil {
		path = Abs(path)
		info, err = du.readFileInfo(path)
	} else {
		fs, ok = source.(vfs.FileSystem)
		if !ok {
			path = Abs(path)
			info, err = du.readFileInfo(path)
		} else {
			info, err = fs.ReadDir(path)
		}
	}

	if err == nil {
		err = du.parseFromVfs(fs, path, info)
	}
	return
}

func (du *Docu) readFileInfo(abs string) ([]os.FileInfo, error) {
	if fi, e := os.Stat(abs); e != nil {
		return nil, e
	} else if !fi.IsDir() {
		return []os.FileInfo{fi}, nil
	}
	fd, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return fd.Readdir(-1)
}

func (du *Docu) parseFromVfs(fs vfs.FileSystem, dir string,
	info []os.FileInfo) (err error) {

	var r vfs.ReadSeekCloser
	if fs == nil {
		fs = vfs.OS(dir)
	}

	for _, info := range info {
		if info.IsDir() || !du.Filter(info.Name()) {
			continue
		}
		if r, err = fs.Open(info.Name()); err == nil {
			err = du.parseFile(dir, info.Name(), r)
			if err == nil {
				err = r.Close()
			} else {
				r.Close()
			}
		}
		if err != nil {
			break
		}
	}
	return
}

func (du *Docu) parseFile(abs, name string, src interface{}) error {
	importPaths := strings.Replace(abs, `\`, `/`, -1)
	pos := strings.LastIndex(importPaths, "/src/")
	if pos != -1 {
		importPaths = importPaths[pos+5:]
	}

	abs = filepath.Join(abs, name)

	astfile, err := parser.ParseFile(du.FileSet, abs, src, parser.ParseComments)
	if err != nil {
		return err
	}
	name = astfile.Name.Name
	// 同目录多包, 比如 main, test
	if path.Base(importPaths) != name {
		importPaths += "." + name
	}
	pkg, ok := du.Astpkg[importPaths]
	if !ok {
		pkg = &ast.Package{
			Name:  name,
			Files: make(map[string]*ast.File),
		}
		du.Astpkg[importPaths] = pkg
	} else if _, ok = pkg.Files[abs]; ok {
		return errors.New("Duplicates: " + abs)
	}
	pkg.Files[abs] = astfile

	return nil
}
