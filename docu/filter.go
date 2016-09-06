package docu

import (
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
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
		var ok, wantType, hasType bool
		for i := len(decl.Specs); i > 0; {
			i--
			if decl.Tok == token.TYPE {
				ok = exportedTypeSpecFilter(decl.Specs[i].(*ast.TypeSpec), by)
			} else {
				ok, hasType = exportedValueSpecFilter(decl.Specs[i].(*ast.ValueSpec), wantType, by)
				if ok {
					wantType = !hasType
				}
			}
			if !ok {
				if i+1 != len(decl.Specs) {
					copy(decl.Specs[i:], decl.Specs[i+1:])
				}
				decl.Specs = decl.Specs[:len(decl.Specs)-1]
			}
		}
		return len(decl.Specs) != 0
	}
	return false
}

func exportedValueSpecFilter(n *ast.ValueSpec, wantType bool, by SortDecl) (bool, bool) {
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
	hasType := n.Type != nil || len(n.Values) != 0
	for i := 0; i < len(n.Names); {
		if n.Names[i].IsExported() || wantType && hasType {
			i++
			continue
		}
		if n.Names[i].Name != "_" {
			spec, _, _ := by.SearchSpec(n.Names[i].Name)
			if spec != nil {
				if _, ok := spec.(*ast.ValueSpec); ok {
					i++
					continue
				}
			}
		}

		if len(n.Names) == len(n.Values) {
			copy(n.Values[i:], n.Values[i+1:])
			n.Values = n.Values[:len(n.Values)-1]
		}
		copy(n.Names[i:], n.Names[i+1:])
		n.Names = n.Names[:len(n.Names)-1]
	}
	hasType = n.Type != nil || len(n.Values) != 0
	return len(n.Names) != 0, hasType
}

func exportedTypeSpecFilter(n *ast.TypeSpec, by SortDecl) bool {
	var bt *ast.StructType
	if !n.Name.IsExported() {
		spec, _, _ := by.SearchSpec(n.Name.String())
		if spec == nil {
			return false
		}
		bySpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			return false
		}
		bt, _ = bySpec.Type.(*ast.StructType)
	}

	st, _ := n.Type.(*ast.StructType)
	exportedFieldFilter(st, bt)
	return true
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

func isExported(name string) bool {
	if strings.IndexByte(name, '.') != -1 {
		return true
	}
	ch, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(ch)
}

// 剔除非导出成员
func exportedFieldFilter(n, by *ast.StructType) {
	if n == nil || n.Fields == nil {
		return
	}
	list := n.Fields.List
	for i := 0; i < len(list); {
		names := list[i].Names
		// 匿名字段
		// type T struct{
		// 	fmt.Stringer
		// }
		if len(names) == 0 {
			if isExported(types.ExprString(list[i].Type)) {
				i++
			} else {
				copy(list[i:], list[i+1:])
				list = list[:len(list)-1]
			}
			continue
		}
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
	_, pos := findField(n.Fields, name)
	return pos != -1
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
	return nil, -1
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

// CanonicalImportPaths 返回 file 注释中的权威导入路径, 如果有的话.
// 返回值是 import 语句中的字符串部分, 含引号.
func CanonicalImportPaths(file *ast.File) string {
	offset := file.Name.Pos() + token.Pos(len(file.Name.String())) + 1
	for _, comm := range file.Comments {
		if comm == nil {
			continue
		}
		at := comm.Pos() - offset
		if at > 0 {
			break
		}
		if at != 0 {
			continue
		}
		if len(comm.List) != 1 || !comm.List[0].Slash.IsValid() ||
			comm.List[0].Text[1] != '/' {
			break
		}
		text := strings.TrimSpace(comm.Text())

		if len(text) > 10 && strings.HasPrefix(text, `import "`) &&
			text[len(text)-1] == '"' {
			return text[7:]
		}
	}
	return ""
}
