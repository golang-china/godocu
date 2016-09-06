// +build go1.5

package docu

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/godoc/vfs"
)

// Docu 复合 token.FileSet, ast.Package 提供 Go doc 支持.
type Docu struct {
	parser.Mode
	FileSet *token.FileSet
	// astpkg 的 key 就是 import paths.
	astpkg map[string]*ast.Package
	// Filter 用于生成 astpkg 时过滤文件名和包名.
	// 显然文件名包含后缀 ".go", 包名则没有.
	Filter func(name string) bool
}

// New 返回使用 DefaultFilter 进行过滤的 Docu 实例.
func New() *Docu {
	return &Docu{parser.ParseComments, token.NewFileSet(),
		make(map[string]*ast.Package), DefaultFilter}
}

// Package 返回 key 对应的 *ast.Package.
// key 为 MergePackageFiles 返回的 paths 元素.
func (du *Docu) Package(key string) *ast.Package {
	if du == nil {
		return nil
	}
	pkg, ok := du.astpkg[key]
	if !ok || pkg == nil {
		return nil
	}
	return pkg
}

// NormalLang 返回 key 对应的 *ast.Package 的 lang.
// 当确定该 pk 符合 Docu 命名风格时使用.
func (du *Docu) NormalLang(key string) string {
	pkg := du.Package(key)
	if pkg == nil || len(pkg.Files) != 1 {
		return ""
	}

	for abs := range pkg.Files {
		base := filepath.Base(abs)
		pos := strings.IndexByte(base, '_') + 1
		end := strings.IndexByte(base, '.')
		if pos == 0 || end <= pos {
			break
		}
		base = base[pos:end]
		if IsNormalLang(base) {
			return base
		}
	}
	return ""
}

// GodocuStyle 用来快速识别是否为 Godocu 文件命名风格.
var GodocuStyle = ast.NewIdent("Godocu Style")

var godocuStyle = []*ast.Ident{GodocuStyle}

func IsGodocuFile(file *ast.File) bool {
	return file != nil && len(file.Unresolved) != 0 &&
		file.Unresolved[0] == GodocuStyle
}

// MergePackageFiles 合并 import paths 的包为一个 ast.File 文件, 并用 Index 排序.
// 如果该 file 是 Godocu 文件命名风格, 设置 file.Unresolved[0] = GodocuStyle.
func (du *Docu) MergePackageFiles(key string) (file *ast.File) {
	if du == nil || len(du.astpkg) == 0 {
		return nil
	}
	pkg, ok := du.astpkg[key]
	if !ok || pkg == nil {
		return
	}
	// 单文件优化
	if len(pkg.Files) == 1 {
		var name string
		for name, file = range pkg.Files {
		}
		if IsNormalName(filepath.Base(name)) {
			file.Unresolved = godocuStyle
		}
	} else {
		// 抛弃无关的 Comments
		file = ast.MergePackageFiles(pkg,
			ast.FilterFuncDuplicates|ast.FilterUnassociatedComments|ast.FilterImportDuplicates)
		// 取出 License 和 import paths 放到 file.Comments
		// 通常 License 总第一个
		var lic, imp *ast.CommentGroup
		for _, f := range pkg.Files {
			offset := f.Name.Pos() + token.Pos(len(file.Name.String())) + 1
			for _, comm := range f.Comments {
				at := comm.Pos() - offset
				if at > 0 {
					break
				}
				if lic == nil && at < 0 {
					text := comm.Text()
					pos := strings.IndexByte(text, ' ')
					if pos != -1 && "copyright" == strings.ToLower(text[:pos]) {
						lic = comm
						continue
					}
				}
				// 简单加入 import paths, 但不检查有效性
				if imp == nil && at == 0 && len(comm.List) == 1 && comm.List[0].Slash.IsValid() {
					file.Package, file.Name = f.Package, f.Name
					imp = comm
					break
				}
			}
			if lic != nil && imp != nil {
				break
			}
		}
		if lic != nil {
			file.Comments = []*ast.CommentGroup{lic}
		}
		if imp != nil {
			file.Comments = append(file.Comments, imp)
		}
	}

	sort.Sort(SortImports(file.Imports))
	Index(file)
	return
}

// IsMultiplePkgError 返回 err 是否为同一个目录下发生多包冲突.
func IsMultiplePkgError(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "multiple packages ")
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
// 返回值 importPaths 从 path 计算得到.
func (du *Docu) Parse(path string, source interface{}) (importPaths string, err error) {
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
		importPaths, err = du.parseFromVfs(fs, path, info)
		return
	}

	// 数据方式
	abs := Abs(path)
	pos := strings.LastIndexAny(abs, `\/`)
	if pos != -1 {
		path, abs = abs[pos+1:], abs[:pos]
	} else {
		path = ""
	}
	importPaths, err = du.parseFile(abs, path, source)

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
	var paths string

	for _, info := range info {
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".go") ||
			!du.filter(info.Name()) {

			continue
		}
		if r, err = fs.Open(info.Name()); err == nil {
			paths, err = du.parseFile(dir, info.Name(), r)
			if err == nil {
				err = r.Close()
			} else {
				r.Close()
			}
			if err == nil && paths != "" {
				if importPaths == "" {
					importPaths = paths
				}
			}
		}
		if err != nil {
			break
		}
	}
	return
}

func (du *Docu) filter(name string) bool {
	return du.Filter == nil || du.Filter(name)
}

func (du *Docu) parseFile(abs, name string, src interface{}) (string, error) {
	var bs []byte
	var err error
	importPaths := LookImportPath(abs)
	if importPaths == "" {
		return "", errors.New("LookImportPath fail: " + abs)
	}
	abs = filepath.Join(abs, name)
	bs, err = readSource(abs, src)
	if err != nil {
		return "", err
	}

	if !IsNormalName(name) && !buildForLinux(bs) {
		return "", nil
	}

	astfile, err := parser.ParseFile(du.FileSet, abs, bs, du.Mode)
	if err != nil {
		return "", err
	}

	name = astfile.Name.String()
	// 包名过滤
	if !du.filter(name) {
		return "", nil
	}

	pkg, ok := du.astpkg[importPaths]
	if !ok {
		pkg = &ast.Package{
			Name:  name,
			Files: make(map[string]*ast.File),
		}
		du.astpkg[importPaths] = pkg
	}

	if ok {
		for oabs, file := range pkg.Files {
			if file.Name.String() != name {
				return importPaths, fmt.Errorf(
					"multiple packages %s (%s) and %s (%s) in %s",
					name, filepath.Base(abs), file.Name.String(), filepath.Base(oabs),
					filepath.Dir(abs))
			}
		}
	}

	if _, ok = pkg.Files[abs]; ok {
		return "", errors.New("Duplicates: " + abs)
	}
	pkg.Files[abs] = astfile

	return importPaths, nil
}
