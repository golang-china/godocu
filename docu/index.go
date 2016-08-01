package docu

import (
	"go/ast"
	"go/token"
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

var numNames = map[int]string{
	ImportNum: "Import",
	ConstNum:  "Const",
	VarNum:    "Var",
	TypeNum:   "Type",
	FuncNum:   "Func",
	MethodNum: "Method",
	OtherNum:  "Other",
}

// NodeNumber 返回值用于节点排序. 随算法更新同类型节点该返回值会变更.
func NodeNumber(node ast.Node) int {
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
	for _, node := range []ast.Decl(s) {
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
	for _, node := range []ast.Decl(s) {
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

// SearchSpec 查找 specIdentLit 对应的 ast.Spec.
func (s SortDecl) SearchSpec(specIdentLit string) ast.Spec {
	if specIdentLit == "" || specIdentLit == "<nil>" {
		return nil
	}
	for _, node := range []ast.Decl(s) {
		switch n := node.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				if SpecIdentLit(spec) == specIdentLit {
					return spec
				}
			}
		}
	}
	return nil
}

// Filter 过滤掉 file 中的非导出顶级声明, 如果 s 中没有该声明的话.
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
//	Imports, Consts, Vars, Types, Funcs, Types.Funcs
//
func Index(file *ast.File) {
	if file != nil {
		sort.Sort(SortDecl(file.Decls))
	}
}

// IndexNormal 对 file 顶级声明进行 normalize 处理.
// 该方法拆分或者合并原声明, 获得更好的排序.
//
// 算法:
//	所有分组顶级声明按类型重新整理分组, 相同类型分为一组.
//	分组以及组内列表按字面值进行排序.
//
func IndexNormal(file *ast.File) {
	//var dest ast.Decl
	// 先进行普通排序
	sort.Sort(SortDecl(file.Decls))
	// 分组合并
	decls := file.Decls
	for _, node := range decls {
		switch n := node.(type) {
		case *ast.GenDecl:
			switch n.Tok {
			case token.IMPORT:
			case token.TYPE:
			case token.CONST, token.VAR:
			}
		case *ast.FuncDecl:
		}
	}
}
