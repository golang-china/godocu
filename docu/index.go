package docu

import (
	"fmt"
	"go/ast"
	"go/token"
	"sort"
)

// NodeNumber 返回值用于节点排序. 随算法更新同类型节点该返回值会变更.
func NodeNumber(node ast.Node) int {
	switch n := node.(type) {
	case *ast.GenDecl:
		switch n.Tok {
		case token.IMPORT:
			return 0
		case token.CONST:
			return 1
		case token.VAR:
			return 2
		case token.TYPE:
			return 4
		}
	case *ast.FuncDecl:
		if n.Recv == nil {
			return 3
		}
		return 5
	}
	// BadDecl 或其他
	return 1 << 32
}

// NodeLit 返回用于排序的 node 字面描述.
func NodeLit(node ast.Node) (lit string) {
	switch n := node.(type) {
	case *ast.GenDecl:
		switch n.Tok {
		case token.IMPORT: // 不排序
		case token.CONST:
		case token.VAR:
		case token.TYPE:
		}
	case *ast.FuncDecl:
		lit = FuncLit(n)
	}
	// BadDecl 或其他
	return
}

// SpecLit 返回 spec 的字面描述.
func SpecLit(spec ast.Spec) (lit string) {
	return
}

// RecvLit 返回返回类型方法接收者 recv 的 Ident 字面描述.
func RecvLit(decl *ast.FuncDecl) (lit string) {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return
	}
	switch expr := decl.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if x, ok := expr.X.(fmt.Stringer); ok {
			lit = "*" + x.String()
		}
	case *ast.Ident:
		lit = expr.String()
	}
	return
}

// FuncLit 返回 FuncDecl 带接收者的字面描述.
func FuncLit(decl *ast.FuncDecl) (lit string) {
	lit = RecvLit(decl)
	if lit == "" {
		lit = decl.Name.String()
	} else {
		lit += "." + decl.Name.String()
	}
	return
}

/*
 * SortDecl 实现 sort.Interface.
 */
type SortDecl []ast.Decl

func (s SortDecl) Len() int { return len(s) }
func (s SortDecl) Less(i, j int) bool {
	in, jn := NodeNumber(s[i]), NodeNumber(s[j])
	if in != jn {
		return in < jn
	}
	switch in {
	case 1:
	case 2:
	case 4:
	case 3, 5:
		return FuncLit(s[i].(*ast.FuncDecl)) < FuncLit(s[j].(*ast.FuncDecl))
	}
	return false
}

func (s SortDecl) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// Index 对 file 顶级声明重新排序. 按照:
//
//	Consts, Types.Consts, Vars, Funcs, Types.Funcs
//
func Index(file *ast.File) {
	sort.Sort(SortDecl(file.Decls))
}

// IndexNormal 对 file 顶级声明进行 normalize 处理.
// 该方法拆分或者合并原声明, 获得更好的排序.
//
// 算法:
//	所有分组顶级声明按类型重新整理分组, 相同类型分为一组.
//	分组以及组内列表按字面值进行排序.
//
func IndexNormal(file *ast.File) {
	sort.Sort(SortDecl(file.Decls))
}
