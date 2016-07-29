// +build go1.5

package docu

import (
	"errors"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
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
	// astpkg 的 key 以 import paths 和包名计算得来.
	// 如果包名为 "main" 或者 "_test" 结尾, key 为 import paths 附加 "::"+包名.
	// 否则 key 为 import paths.
	astpkg map[string]*ast.Package
	// Filter 用于生成 astpkg 时过滤目录名和文件名 name.
	// name 不包括上级路径.
	Filter func(name string) bool
}

// New 返回使用 DefaultFilter 进行过滤的 Docu.
func New(mode Mode) *Docu {
	du := &Docu{mode, parser.ParseComments, token.NewFileSet(), make(map[string]*ast.Package), DefaultFilter}
	if mode|ShowTest != 0 {
		du.Filter = ShowTestFilter
	}
	return du
}

func (du *Docu) Package(key string) *ast.Package {
	pkg, ok := du.astpkg[key]
	if !ok || pkg == nil {
		return nil
	}
	return pkg
}

// MergePackageFiles 合并 import paths 的包为一个已排序的 ast.File 文件.
func (du *Docu) MergePackageFiles(paths string) (file *ast.File) {
	pkg, ok := du.astpkg[paths]
	if !ok || pkg == nil {
		return
	}

	file = ast.MergePackageFiles(pkg,
		ast.FilterFuncDuplicates|ast.FilterUnassociatedComments|ast.FilterImportDuplicates)

	// 取出 License 放到 file.Comments
	for _, f := range pkg.Files {
		for _, comm := range f.Comments {
			lic := comm.Text()
			if doc.Synopsis(lic) == "" &&
				"copyright" == strings.ToLower(strings.SplitN(lic, " ", 2)[0]) {
				file.Comments = []*ast.CommentGroup{comm}
				break
			}
			lic = ""
		}
		if len(file.Comments) != 0 {
			break
		}
	}

	sort.Sort(SortImports(file.Imports))
	Index(file)
	return
}

// Parse 解析 path,source 并返回本次解析到的包路径和发生的错误.
//
//  应预先格式化 path,source 组合对应的代码.
//  如果无法确定文件名将产生序号文件名替代.
//
// path:
//   import paths 或 Go 文件名
// source:
//   nil
//   vfs.FileSystem
//   []byte,string,io.Reader,*bytes.Buffer
//
func (du *Docu) Parse(path string, source interface{}) (paths []string, err error) {
	var info []os.FileInfo
	var fs vfs.FileSystem
	var ok bool

	if source == nil {
		path = Abs(path)
		info, err = du.readFileInfo(path)
		if err == errIsFile {
			err = nil
			path = path[:len(path)-len(info[0].Name())]
		}
		fs = vfs.OS(path)
	} else if fs, ok = source.(vfs.FileSystem); ok {
		info, err = fs.ReadDir(path)
	}

	if err != nil {
		return
	}

	if fs != nil {
		path, err = du.parseFromVfs(fs, path, info)
		if path != "" {
			paths = strings.Split(path, "\n")
			sort.Strings(paths)
		}
		return
	}

	// 数据方式
	abs := Abs(path)
	if !du.Filter(path) {
		return nil, errors.New("Parse: invalid path: " + path)
	}
	pos := strings.LastIndexAny(abs, `\/`)
	if pos != -1 {
		path, abs = abs[pos+1:], abs[:pos]
	} else {
		path = ""
	}
	path, err = du.parseFile(abs, path, source)
	if path != "" {
		paths = strings.Split(path, "\n")
		sort.Strings(paths)
	}

	return
}

var errIsFile = errors.New("")

func (du *Docu) readFileInfo(abs string) ([]os.FileInfo, error) {
	if fi, e := os.Stat(abs); e != nil {
		return nil, e
	} else if !fi.IsDir() {
		return []os.FileInfo{fi}, errIsFile
	}
	fd, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return fd.Readdir(-1)
}

func (du *Docu) parseFromVfs(fs vfs.FileSystem, dir string,
	info []os.FileInfo) (importPaths string, err error) {

	var r vfs.ReadSeekCloser
	var s string

	importPaths = nl
	for _, info := range info {
		if info.IsDir() || !du.Filter(info.Name()) {
			continue
		}
		if r, err = fs.Open(info.Name()); err == nil {
			s, err = du.parseFile(dir, info.Name(), r)
			if s != "" && strings.Index(importPaths, nl+s+nl) == -1 {
				importPaths += s + nl
			}
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
	if importPaths == nl {
		importPaths = ""
	} else {
		importPaths = importPaths[1 : len(importPaths)-1]
	}
	return
}

func (du *Docu) parseFile(abs, name string, src interface{}) (string, error) {
	none := name == ""
	importPaths := strings.Replace(abs, `\`, `/`, -1)
	abs = filepath.Join(abs, name)

	pos := strings.LastIndex(importPaths, "/src/")
	if pos != -1 {
		importPaths = importPaths[pos+5:]
	}

	astfile, err := parser.ParseFile(du.FileSet, abs, src, du.parserMode)
	if err != nil {
		return "", err
	}

	name = astfile.Name.String()
	if du.mode&ShowCMD == 0 && name == "main" {
		return "", nil
	}
	if du.mode&ShowTest == 0 && (name == "test" || strings.HasSuffix(name, "_test")) {
		return "", nil
	}

	if du.mode&ShowUnexported == 0 && !ExportedFileFilter(astfile) {
		// 虽然可能没有导出内容, 但是可能有文档
		if astfile.Doc == nil || len(astfile.Doc.List) == 0 {
			return "", nil
		}
	}

	// 同目录多包, 比如 main, test
	if name == "main" || name == "test" || strings.HasSuffix(name, "_test") {
		importPaths += "::" + name
	}
	pkg, ok := du.astpkg[importPaths]
	if !ok {
		pkg = &ast.Package{
			Name:  name,
			Files: make(map[string]*ast.File),
		}
		du.astpkg[importPaths] = pkg
	}
	if none {
		abs = filepath.Join(abs, "_"+strconv.Itoa(len(pkg.Files))+".go")
	}
	if _, ok = pkg.Files[abs]; ok {
		return importPaths, errors.New("Duplicates: " + abs)
	}
	pkg.Files[abs] = astfile

	return importPaths, nil
}
