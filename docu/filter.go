package docu

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var DefaultFilter = PackageFilter

// PackageFilter 过滤掉 "main","test" 相关包, 过滤掉非 ".go" 文件.
// 使用时应先过滤掉非 ".go" 的文件, 该函数才能正确工作
func PackageFilter(name string) bool {
	if name == "" || name[0] == '_' || name[0] == '.' ||
		name == "main" || name == "test" || name == "main.go" || name == "test.go" {

		return false
	}

	if strings.HasSuffix(name, ".go") {
		return !strings.HasPrefix(name, "test_") &&
			!strings.HasPrefix(name, "main_") &&
			!strings.HasSuffix(name, "_test.go")
	}
	return strings.IndexByte(name, '.') == -1
}

// TestFilter 过滤掉非 "test" 相关的包和文件
func TestFilter(name string) bool {
	if name == "" || name[0] == '_' || name[0] == '.' {
		return false
	}
	if strings.HasSuffix(name, ".go") {
		return name == "test.go" || strings.HasSuffix(name, "_test.go")
	}
	return name != "main"
}

// MainFilter 过滤掉非 "main" 相关的包和文件
func MainFilter(name string) bool {
	if name == "" || name[0] == '_' || name[0] == '.' {
		return false
	}
	if strings.HasSuffix(name, ".go") {
		return name != "test.go" && !strings.HasSuffix(name, "_test.go") &&
			!strings.HasPrefix(name, "test_") &&
			!strings.HasPrefix(name, "doc_")
	}

	return name == "main"
}

// GenNameFilter 返回一个过滤函数, 允许所有包名或 go 源文件名等于 filename 通过
func GenNameFilter(filename string) func(string) bool {
	return func(name string) bool {
		if name == "" || name[0] == '_' || name[0] == '.' {
			return false
		}
		return filename == name || !strings.HasSuffix(name, ".go")
	}
}

// ExportedFileFilter 剔除 non-nil file 中所有非导出声明, 返回该 file 是否具有导出声明.
func ExportedFileFilter(file *ast.File) bool {
	return exportedFileFilter(file, nil)
}

func exportedFileFilter(file *ast.File, by SortDecl) bool {
	max := len(file.Decls)
	for i := 0; i < max; {
		if exportedDeclFilter(file.Decls[i], by) {
			i++
			continue
		}
		copy(file.Decls[i:], file.Decls[i+1:max])
		max--
	}
	file.Decls = file.Decls[:max]
	return max != 0
}

// ExportedDeclFilter 剔除 non-nil decl 中所有非导出声明, 返回该 decl 是否具有导出声明.
func ExportedDeclFilter(decl ast.Decl) bool {
	return exportedDeclFilter(decl, nil)
}

func exportedDeclFilter(decl ast.Decl, by SortDecl) bool {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		if decl.Recv != nil && !exportedRecvFilter(decl.Recv, by) {
			return false
		}
		return decl.Name.IsExported() || nil != by.SearchFunc(FuncIdentLit(decl))
	case *ast.GenDecl:
		for i := 0; i < len(decl.Specs); {
			if exportedSpecFilter(decl.Specs[i], by) {
				i++
				continue
			}
			copy(decl.Specs[i:], decl.Specs[i+1:])
			decl.Specs = decl.Specs[:len(decl.Specs)-1]
		}
		return len(decl.Specs) != 0
	}
	return false
}

// exportedRecvFilter 该方法仅仅适用于检测 ast.FuncDecl.Recv 是否导出
func exportedRecvFilter(fieldList *ast.FieldList, by SortDecl) bool {

	for i := 0; i < len(fieldList.List); i++ {
		switch n := fieldList.List[i].Type.(type) {
		case *ast.Ident:
			if spec, _, _ := by.SearchSpec(n.String()); !n.IsExported() && spec == nil {
				return false
			}
		case *ast.StarExpr:
			ident, ok := n.X.(*ast.Ident)
			if !ok {
				return false
			}
			if spec, _, _ := by.SearchSpec(ident.String()); !ident.IsExported() && spec == nil {
				return false
			}
		}
	}
	return true
}

// ExportedSpecFilter 剔除 non-nil spec 中所有非导出声明, 返回该 spec 是否具有导出声明.
func ExportedSpecFilter(spec ast.Spec) bool {
	return exportedSpecFilter(spec, nil)
}

func exportedSpecFilter(spec ast.Spec, by SortDecl) bool {
	switch n := spec.(type) {
	case *ast.ImportSpec:
		return true
	case *ast.ValueSpec:
		for i := 0; i < len(n.Names); {
			// 特别情况:
			//	const (
			//		_ Mode = iota
			//		ModeARM
			//		ModeThumb
			//	)
			//
			// 和 syscall SOL_SOCKET
			// const (
			// 		_ = iota
			// 		SOL_SOCKET
			// )
			if n.Names[i].IsExported() ||
				i == 0 && len(n.Names) == 1 && n.Names[0].Name == "_" {
				i++
				continue
			}
			spec, _, _ := by.SearchSpec(SpecIdentLit(n))
			if spec != nil {
				if bySpec, _ := spec.(*ast.ValueSpec); bySpec != nil {
					i++
					continue
				}
			}

			copy(n.Names[i:], n.Names[i+1:])
			n.Names = n.Names[:len(n.Names)-1]
		}
		return len(n.Names) != 0
	case *ast.TypeSpec:
		spec, _, _ := by.SearchSpec(SpecIdentLit(n))
		if !n.Name.IsExported() && spec == nil {
			return false
		}
		bySpec, _ := spec.(*ast.TypeSpec)
		if !n.Name.IsExported() && bySpec == nil {
			return false
		}

		var bt *ast.StructType

		st, _ := n.Type.(*ast.StructType)
		if bySpec != nil {
			bt, _ = bySpec.Type.(*ast.StructType)
		}
		exportedFieldFilter(st, bt)
	}
	return true
}

// 剔除非导出成员
func exportedFieldFilter(n, by *ast.StructType) {
	if n == nil || n.Fields == nil {
		return
	}
	list := n.Fields.List
	for i := 0; i < len(list); {
		names := list[i].Names
		for i := 0; i < len(names); {
			if names[i].IsExported() || by != nil &&
				hasField(by, names[i].String()) {
				i++
				continue
			}
			copy(names[i:], names[i+1:])
			names = names[:len(names)-1]
		}
		list[i].Names = names
		if len(names) != 0 {
			i++
			continue
		}
		copy(list[i:], list[i+1:])
		list = list[:len(list)-1]
	}
	n.Fields.List = list
	return
}

func hasField(n *ast.StructType, name string) bool {
	if n == nil || n.Fields == nil {
		return false
	}

	for _, field := range n.Fields.List {
		if field == nil {
			continue
		}
		for _, ident := range field.Names {
			if ident.String() == name {
				return true
			}
		}
	}
	return false
}

func findField(n *ast.FieldList, lit string) (*ast.Field, int) {
	for _, field := range n.List {
		if field == nil {
			continue
		}
		for i, ident := range field.Names {
			if ident.String() == lit {
				return field, i
			}
		}
	}
	return nil, 0
}

// WalkPath 可遍历 paths 及其子目录或者独立的文件.
// 若 paths 是包路径或绝对路径, 调用 walkFn 遍历 paths.
func WalkPath(paths string, walkFn filepath.WalkFunc) error {
	root := Abs(paths)
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() {
		return walkFn(root, info, err)
	}

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		ispkg := IsPkgDir(info)
		if err == nil && info.IsDir() {
			if info.Name() == "src" {
				return nil
			}
			if !ispkg {
				return filepath.SkipDir
			}
		}
		if err != nil || ispkg {
			return walkFn(path, info, err)
		}
		return nil
	})
}

func walkPath(path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(path, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := readDirNames(path)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := os.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walkPath(filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	return nil
}

func readDirNames(dirname string) ([]string, error) {
	f, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func IsPkgDir(fi os.FileInfo) bool {
	name := fi.Name()
	return fi.IsDir() && len(name) > 0 &&
		name[0] != '_' && name[0] != '.' &&
		name != "testdata" && name != "vendor"
}

// License 返回 file 中以 copyright 开头的注释, 如果有的话.
func License(file *ast.File) (lic string) {
	for _, comm := range file.Comments {
		lic = comm.Text()
		pos := strings.IndexByte(lic, ' ')
		if pos != -1 && "copyright" == strings.ToLower(lic[:pos]) {
			return lic
		}
	}
	return ""
}

// CanonicalImportPaths 返回 file 中的权威导入路径注释, 如果有的话.
func CanonicalImportPaths(file *ast.File) string {
	offset := file.Name.Pos() + token.Pos(len(file.Name.String())) + 1
	for _, comm := range file.Comments {
		at := comm.Pos() - offset
		if at > 0 {
			break
		}
		if at != 0 {
			continue
		}
		if len(comm.List) != 1 || !comm.List[0].Slash.IsValid() {
			break
		}
		text := strings.TrimSpace(comm.Text())
		paths := strings.Split(text, " ")
		if len(paths) == 2 && paths[0] == "import" &&
			paths[1][0] == '"' && paths[1][len(paths[1])-1] == '"' {
			return text
		}
	}
	return ""
}
