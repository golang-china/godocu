package docu

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// SuffixFilter 返回文件名后缀过滤函数. 例如 SuffixFilter("_zh_CN.go").
func SuffixFilter(suffix string) func(string) bool {
	return func(name string) bool {
		return DefaultFilter(name) && strings.HasSuffix(name, suffix)
	}
}

// DefaultFilter 缺省的文件名过滤规则. 过滤掉非 ".go" 和 "_test.go" 结尾的文件
func DefaultFilter(name string) bool {
	return ShowTestFilter(name) && !strings.HasSuffix(name, "_test.go")
}

// ShowTestFilter 允许 "_test.go" 结尾的文件
func ShowTestFilter(name string) bool {
	return len(name) > 0 && name[0] != '_' && name[0] != '.' && strings.HasSuffix(name, ".go")
}

// ExportedFileFilter 剔除 non-nil file 中所有非导出声明, 返回该 file 是否具有导出声明.
func ExportedFileFilter(file *ast.File) bool {
	return exportedFileFilter(file, nil)
}

func exportedFileFilter(file *ast.File, by SortDecl) bool {
	for i := 0; i < len(file.Decls); {
		if exportedDeclFilter(file.Decls[i], by) {
			i++
			continue
		}
		copy(file.Decls[i:], file.Decls[i+1:])
		file.Decls = file.Decls[:len(file.Decls)-1]
	}
	return len(file.Decls) != 0
}

// ExportedDeclFilter 剔除 non-nil decl 中所有非导出声明, 返回该 decl 是否具有导出声明.
func ExportedDeclFilter(decl ast.Decl) bool {
	return exportedDeclFilter(decl, nil)
}

func exportedDeclFilter(decl ast.Decl, by SortDecl) bool {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		//  写法复杂, 为了效率
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
			if !n.IsExported() && nil == by.SearchSpec(n.String()) {
				return false
			}
		case *ast.StarExpr:
			ident, ok := n.X.(*ast.Ident)
			if !ok || !ident.IsExported() && nil == by.SearchSpec(ident.String()) {
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
			if n.Names[i].IsExported() {
				i++
				continue
			}
			bySpec, _ := by.SearchSpec(SpecIdentLit(n)).(*ast.ValueSpec)
			if bySpec != nil {
				i++
				continue
			}

			copy(n.Names[i:], n.Names[i+1:])
			n.Names = n.Names[:len(n.Names)-1]
		}
		return len(n.Names) != 0
	case *ast.TypeSpec:
		bySpec, _ := by.SearchSpec(n.Name.String()).(*ast.TypeSpec)
		if !n.Name.IsExported() && nil == bySpec {
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
			if names[i].IsExported() || by == nil ||
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

// WalkPath 可遍历 paths 及其子目录或者独立的文件.
// 若 paths 是包路径或绝对路径, 调用 walk 遍历 paths.
func WalkPath(paths string, walk filepath.WalkFunc) error {
	root := Abs(paths)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return walk(root, info, err)
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
			return walk(path, info, err)
		}
		return nil
	})
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

// ImportPaths 返回 file 中的权威导入路径注释, 如果有的话.
func ImportPaths(file *ast.File) string {
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
