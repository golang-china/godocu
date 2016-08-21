package docu

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"
)

// DeclIdentLit 返回返 decl 第一个 ast.Spec 的 Ident 字面描述.
// 如果是 method, 返回风格为 RecvIdentLit.FuncName.
// 如果是 GenDecl 只返回第一个 Spec 的 Ident 字面描述
func DeclIdentLit(decl ast.Decl) (lit string) {
	switch n := decl.(type) {
	case *ast.GenDecl:
		if len(n.Specs) != 0 {
			return SpecIdentLit(n.Specs[0])
		}
	case *ast.FuncDecl:
		return FuncIdentLit(n)
	}
	return
}

// SpecIdentLit 返回 spec 首个 Ident 字面描述.
func SpecIdentLit(spec ast.Spec) (lit string) {
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

// SpecTypeIdentLit 返回 spec 类型 Ident 字面描述.
func SpecTypeLit(spec ast.Spec) (lit string) {
	switch n := spec.(type) {
	case *ast.ValueSpec:
		lit = types.ExprString(n.Type)
	case *ast.ImportSpec:
	case *ast.TypeSpec:
		lit = types.ExprString(n.Type)
	}
	return
}

func SpecDoc(spec ast.Spec) *ast.CommentGroup {
	if spec == nil {
		return nil
	}
	switch n := spec.(type) {
	case *ast.ValueSpec:
		return n.Doc
	case *ast.ImportSpec:
		return n.Doc
	case *ast.TypeSpec:
		return n.Doc
	}
	return nil
}

// RecvIdentLit 返回返回类型方法接收者 recv 的 Ident 字面描述.
func RecvIdentLit(decl *ast.FuncDecl) (lit string) {
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

// FuncIdentLit 返回 FuncDecl 的 Ident 字面描述.
// 如果是 method, 返回风格为 RecvIdentLit.FuncName.
func FuncIdentLit(decl *ast.FuncDecl) (lit string) {
	lit = RecvIdentLit(decl)
	if lit == "" {
		return decl.Name.String()
	}
	return lit + "." + decl.Name.String()
}

// FuncLit 返回 FuncDecl 的字面描述.
func FuncLit(decl *ast.FuncDecl) (lit string) {
	lit = FieldListLit(decl.Type.Params)
	suffix := FieldListLit(decl.Type.Results)
	if suffix == "" {
		suffix = "(" + lit + ")"
	} else if strings.IndexAny(suffix, " ,") == -1 {
		suffix = "(" + lit + ") " + suffix
	} else {
		suffix = "(" + lit + ") (" + suffix + ")"
	}

	if decl.Name != nil {
		suffix = decl.Name.String() + suffix
	}

	lit = RecvIdentLit(decl)
	if lit == "" {
		lit = "func " + suffix
	} else {
		lit = "func (" + lit + ") " + suffix
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

// ImportsString 返回 imports 源码.
func ImportsString(is []*ast.ImportSpec) (s string) {
	if len(is) == 0 {
		return
	}

	if len(is) == 1 {
		return "import " + is[0].Path.Value + nl
	}
	for i, im := range is {
		if i == 0 {
			s += "import (\n    " + im.Path.Value + nl
		} else {
			s += "    " + im.Path.Value + nl
		}
	}
	s += ")\n"
	return
}
