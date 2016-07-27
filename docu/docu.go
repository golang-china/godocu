// +build go1.5

package docu

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/tools/godoc/vfs"
)

type Mode int

const (
	ShowCMD Mode = 1 << iota
	ShowUnexported
	ShowTest
)

// Docu 复合 token.FileSet, ast.Package 提供 Go doc 支持.
type Docu struct {
	mode       Mode
	parserMode parser.Mode
	*token.FileSet
	// Astpkg 的 key 以 import paths 和包名计算得来.
	// 如果包名为 "main" 或者 "_test" 结尾, key 为 import paths 附加 ":"+包名.
	// 否则 key 为 import paths.
	Astpkg map[string]*ast.Package
	// Filter 用于生成 Astpkg 时过滤目录名和文件名 name.
	// name 不包括上级路径.
	Filter func(name string) bool
}

// New 返回使用 DefaultFilter 进行过滤的 Docu.
func New(mode Mode) Docu {
	du := Docu{mode, parser.ParseComments, token.NewFileSet(), make(map[string]*ast.Package), DefaultFilter}
	if mode|ShowTest != 0 {
		du.Filter = ShowTestFilter
	}
	return du
}

// MergePackageFiles 合并 du.Astpkg[key] 为一个 ast.File 文件.
// 并对 import path 进行排序.
func (du *Docu) MergePackageFiles(key string) *ast.File {
	pkg, ok := du.Astpkg[key]
	if !ok || pkg == nil {
		return nil
	}
	file := ast.MergePackageFiles(pkg,
		ast.FilterFuncDuplicates|ast.FilterUnassociatedComments|ast.FilterImportDuplicates)
	sort.Sort(SortImports(file.Imports))
	return file
}

// Parse 解析 path,source 并返回发生的错误.
//
// 	要求 path,source 组合后对应的代码都已格式化.
//  如果无法确定文件名将生产临时文件名于 FileSet.
//
// path:
//   import paths 或 Go 文件名
// source:
//   nil
//   vfs.FileSystem
//   []byte,string,io.Reader,*bytes.Buffer
//
func (du *Docu) Parse(path string, source interface{}) (err error) {
	var info []os.FileInfo
	var fs vfs.FileSystem
	var ok bool

	if source == nil {
		path = Abs(path)
		info, err = du.readFileInfo(path)
	} else if fs, ok = source.(vfs.FileSystem); ok {
		info, err = fs.ReadDir(path)
	} else {
		// 文件方式
		abs := Abs(path)
		if !du.Filter(path) {
			return errors.New("invalid path: " + path)
		}
		pos := strings.LastIndexAny(abs, `\/`)
		if pos != -1 {
			path, abs = abs[pos+1:], abs[:pos]
		} else {
			path = ""
		}
		return du.parseFile(abs, path, source)
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
	none := name == ""
	importPaths := path.Clean(abs)
	if src == nil {
		abs = filepath.Join(abs, name)
	} else {
		abs = path.Join(importPaths, name)
	}

	pos := strings.LastIndex(importPaths, "/src/")
	if pos != -1 {
		importPaths = importPaths[pos+5:]
	}

	astfile, err := parser.ParseFile(du.FileSet, abs, src, du.parserMode)
	if err != nil {
		return err
	}

	name = astfile.Name.String()
	if du.mode&ShowCMD == 0 && name == "main" {
		return nil
	}
	if du.mode&ShowTest == 0 && (name == "test" || strings.HasSuffix(name, "_test")) {
		return nil
	}

	if du.mode&ShowUnexported == 0 {
		// 虽然可能没有导出内容, 但是可能有文档
		ExportedFileFilter(astfile)
	}

	// 同目录多包, 比如 main, test
	if name == "main" || name == "test" || strings.HasSuffix(name, "_test") {
		importPaths += ":" + name
	}
	pkg, ok := du.Astpkg[importPaths]
	if !ok {
		pkg = &ast.Package{
			Name:  name,
			Files: make(map[string]*ast.File),
		}
		du.Astpkg[importPaths] = pkg
	}
	if none {
		if src == nil {
			abs = filepath.Join(abs, "_"+strconv.Itoa(len(pkg.Files))+".go")
		} else {
			abs = path.Join(abs, "_"+strconv.Itoa(len(pkg.Files))+".go")
		}
	}
	if _, ok = pkg.Files[abs]; ok {
		return errors.New("Duplicates: " + abs)
	}
	pkg.Files[abs] = astfile

	return nil
}
