package docu

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"
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

// NodeLit 返回用于排序的 node 字面描述.
func NodeLit(node ast.Node) (lit string) {
	switch n := node.(type) {
	case *ast.GenDecl:
		switch n.Tok {
		case token.IMPORT, token.CONST, token.VAR, token.TYPE:
			if len(n.Specs) != 0 {
				lit = SpecLit(n.Specs[0])
			}
		}
	case *ast.FuncDecl:
		lit = FuncLit(n)
	}
	// BadDecl 或其他
	return
}

// SpecsLit 返回  SpecLit(specs[0])
func SpecsLit(specs []ast.Spec) (lit string) {
	if len(specs) != 0 {
		lit = SpecLit(specs[0])
	}
	return
}

// SpecLit 返回 spec 的字面描述.
func SpecLit(spec ast.Spec) (lit string) {
	switch n := spec.(type) {
	case *ast.ValueSpec:
		if len(n.Names) != 0 {
			lit = n.Names[0].String()
		}
	case *ast.ImportSpec:
		lit = n.Path.Value
	case *ast.TypeSpec:
		lit = n.Name.String()
	}
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
		if decl.Name != nil {
			lit = decl.Name.String()
		} else {
			results := FieldListLit(decl.Type.Results)
			if strings.IndexByte(results, ',') != -1 {
				results = " (" + results + ")"
			}
			lit = "func (" +
				FieldListLit(decl.Type.Params) +
				")" + results
		}
	} else {
		lit += "." + decl.Name.String()
	}
	return
}

// FieldListLit 返回 ast.FieldList.List 的字面值.
// 该方法仅适用于:
//	ast.FuncDecl.Type.Params
//	ast.FuncDecl.Type.Results
//
func FieldListLit(list *ast.FieldList) (lit string) {
	if list == nil || len(list.List) == 0 {
		return
	}
	for i, field := range list.List {
		if i != 0 {
			lit += ", "
		}
		lit += FieldLit(field)
	}
	return
}

// FieldLit 返回 ast.Field 的字面值
// 该方法与 FieldListLit 配套使用.
func FieldLit(field *ast.Field) (lit string) {
	if field == nil {
		return
	}
	for i, name := range field.Names {
		if i == 0 {
			lit = name.String()
		} else {
			lit += ", " + name.String()
		}
	}
	if field.Type != nil {
		if lit == "" {
			lit = types.ExprString(field.Type)
		} else {
			lit += " " + types.ExprString(field.Type)
		}
	}
	return
}

// SpecsLit 返回  SpecTypeLit(specs[0])
func SpecsTypeLit(specs []ast.Spec) (lit string) {
	if len(specs) != 0 {
		lit = SpecTypeLit(specs[0])
	}
	return
}

// SpecTypeLit 返回 spec.Type 的字面描述.
func SpecTypeLit(spec ast.Spec) (lit string) {
	if spec == nil {
		return
	}
	switch n := spec.(type) {
	case *ast.ValueSpec:
		switch expr := n.Type.(type) {
		case *ast.Ident:
			lit = expr.Name
		case *ast.BasicLit:
			lit = expr.Kind.String()
		case *ast.FuncLit:
			lit = "func"
		}
	case *ast.ImportSpec:
		// ""
	case *ast.TypeSpec:
		lit = n.Name.String()
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
	default:
		si := s[i].(*ast.GenDecl).Specs
		sj := s[j].(*ast.GenDecl).Specs
		if len(si) == 0 || len(sj) == 0 {
			break
		}
		switch in {
		case ConstNum, VarNum: // const, var
			return SpecLit(si[0]) < SpecLit(sj[0])
		case TypeNum: // type
			return si[0].(*ast.TypeSpec).Name.String() <
				sj[0].(*ast.TypeSpec).Name.String()
		}
	case FuncNum, MethodNum:
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
