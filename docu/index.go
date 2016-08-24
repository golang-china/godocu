package docu

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

// 下列常量用于排序
const (
	ImportNum int = iota
	ConstNum
	VarNum
	TypeNum
	FuncNum
	MethodNum
	OtherNum = 1 << 32
)

var numNames = []string{
	ImportNum: "import ",
	ConstNum:  "const ",
	VarNum:    "var ",
	TypeNum:   "type ",
	FuncNum:   "func ",
}

// NodeNumber 返回值用于节点排序. 随算法更新同类型节点该返回值会变更.
func NodeNumber(node ast.Node) int {
	if node == nil {
		return OtherNum
	}
	switch n := node.(type) {
	case *ast.GenDecl:
		switch n.Tok {
		case token.IMPORT:
			return ImportNum
		case token.CONST:
			return ConstNum
		case token.VAR:
			return VarNum
		case token.TYPE:
			return TypeNum
		}
	case *ast.FuncDecl:
		if n.Recv == nil {
			return FuncNum
		}
		return MethodNum
	}
	// BadDecl 或其他
	return OtherNum
}

/*
 * SortDecl 实现 sort.Interface. 按 Ident字面值排序.
 */
type SortDecl []ast.Decl

func (s SortDecl) Len() int      { return len(s) }
func (s SortDecl) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortDecl) Less(i, j int) bool {
	in, jn := NodeNumber(s[i]), NodeNumber(s[j])
	if in != jn {
		return in < jn
	}
	switch in {
	default:
		si := s[i].(*ast.GenDecl).Specs
		sj := s[j].(*ast.GenDecl).Specs
		if len(si) == 0 || len(sj) == 0 {
			break
		}
		return SpecIdentLit(si[0]) < SpecIdentLit(sj[0])
	case FuncNum, MethodNum:
		return FuncLit(s[i].(*ast.FuncDecl)) < FuncLit(s[j].(*ast.FuncDecl))
	}
	return false
}

// Search 查找 identLit 所在的顶级声明.
func (s SortDecl) Search(identLit string) ast.Decl {
	if identLit == "" || identLit == "<nil>" {
		return nil
	}
	for _, node := range s {
		switch n := node.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				if SpecIdentLit(spec) == identLit {
					return node
				}
			}
		case *ast.FuncDecl:
			lit := FuncIdentLit(n)
			if lit == identLit {
				return node
			}
			if lit > identLit {
				break
			}
		default:
			break
		}
	}
	return nil
}

// SearchFunc 查找 funcIdentLit 对应的 ast.FuncDecl.
func (s SortDecl) SearchFunc(funcIdentLit string) *ast.FuncDecl {
	if funcIdentLit == "" || funcIdentLit == "<nil>" {
		return nil
	}
	for _, node := range s {
		switch n := node.(type) {
		case *ast.FuncDecl:
			lit := FuncIdentLit(n)
			if lit == funcIdentLit {
				return n
			}
			if lit > funcIdentLit {
				break
			}
		}
	}
	return nil
}

// SearchConstructor 搜索并返回 typeLit 的构建函数声明:
// 	*typeLit
// 	*typeLit, error
// 	typeLit
// 	typeLit, error
func (s SortDecl) SearchConstructor(typeLit string) *ast.FuncDecl {
	if typeLit == "" || typeLit == "<nil>" {
		return nil
	}
	for _, node := range s {
		if node == nil {
			continue
		}
		switch n := node.(type) {
		case *ast.FuncDecl:
			if isConstructor(n, typeLit) {
				return n
			}
		}
	}
	return nil
}

func isConstructor(n *ast.FuncDecl, typeLit string) bool {
	if !n.Name.IsExported() || n.Type.Results == nil {
		return false
	}
	list := n.Type.Results.List
	switch len(list) {
	case 2:
		if types.ExprString(list[1].Type) != "error" {
			break
		}
		fallthrough
	case 1:
		lit := types.ExprString(list[0].Type)
		if lit == typeLit || lit[0] == '*' && lit[1:] == typeLit {
			return true
		}
	}
	return false
}

// SearchSpec 查找 specIdentLit 对应的顶级 ast.Spec 和所在 *ast.GenDecl 以及索引.
func (s SortDecl) SearchSpec(specIdentLit string) (ast.Spec, *ast.GenDecl, int) {
	if specIdentLit == "" || specIdentLit == "<nil>" {
		return nil, nil, -1
	}
	for _, node := range s {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				break
			}
			for i, spec := range n.Specs {
				if SpecIdentLit(spec) == specIdentLit {
					return spec, n, i
				}
			}
		}
	}
	return nil, nil, -1
}

// Filter 过滤掉 file 中的非导出顶级声明, 如果该声明不在 s 中的话.
// imports 声明总是被保留.
func (s SortDecl) Filter(file *ast.File) bool {
	return exportedFileFilter(file, s)
}

// SortImports 实现 sort.Interface. 按照 import path 进行排序.
type SortImports []*ast.ImportSpec

func (s SortImports) Len() int      { return len(s) }
func (s SortImports) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s SortImports) Less(i, j int) bool {
	return s[i].Path.Value < s[j].Path.Value
}

// Index 对 file 顶级声明重新排序. 按照:
//
//	Imports, Consts, Vars, Types, Funcs, Method
//
func Index(file *ast.File) {
	if file != nil {
		sort.Sort(SortDecl(file.Decls))
	}
}

// IndexNormal 对 file 顶级声明进行常规习惯排序, 即:
//
//	Imports, Consts, Vars, Funcs, Types [Constructor,Method]
//
func IndexNormal(file *ast.File) {
	panic("Unimplemented")
	sort.Sort(sortNormal(file.Decls))
}

var normalOrder = [...]int{
	ImportNum,
	ConstNum,
	VarNum,
	FuncNum,
	TypeNum,
	MethodNum,
}

// sortNormal 实现 sort.Interface. 按常规习惯排序.
type sortNormal []ast.Decl

func (s sortNormal) Len() int      { return len(s) }
func (s sortNormal) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s sortNormal) Less(i, j int) bool {
	in, jn := NodeNumber(s[i]), NodeNumber(s[j])
	if in == OtherNum || jn == OtherNum ||
		in != jn && (in <= VarNum || jn <= VarNum) {
		return in < jn
	}

	if in != jn && (in == FuncNum || jn == FuncNum) {
		return normalOrder[in] < normalOrder[jn]
	}
	// TypeNum, MethodNum
	if in == jn {
		switch in {
		case TypeNum:
			si := s[i].(*ast.GenDecl).Specs
			sj := s[j].(*ast.GenDecl).Specs
			if len(si) == 0 || len(sj) == 0 {
				break
			}
			return SpecIdentLit(si[0]) < SpecIdentLit(sj[0])
		case MethodNum:
			return FuncLit(s[i].(*ast.FuncDecl)) < FuncLit(s[j].(*ast.FuncDecl))
		}
	}
	return false
}
