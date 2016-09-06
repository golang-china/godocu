package docu

import (
	"fmt"
	"go/ast"
	"go/types"
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
	case *ast.TypeSpec:
		lit = types.ExprString(n.Type)
	}
	return
}

// SpecDoc 返回 spec 的 Doc,Comment 字段
func SpecDoc(spec ast.Spec) *ast.CommentGroup {
	if spec == nil {
		return nil
	}
	switch n := spec.(type) {
	case *ast.ValueSpec:
		return n.Doc
	case *ast.TypeSpec:
		return n.Doc
	}
	return nil
}

// SpecComment 返回 spec 的 Doc,Comment 字段
func SpecComment(spec ast.Spec) (*ast.CommentGroup, *ast.CommentGroup) {
	if spec == nil {
		return nil, nil
	}
	switch n := spec.(type) {
	case *ast.ValueSpec:
		return n.Doc, n.Comment
	case *ast.TypeSpec:
		return n.Doc, n.Comment
	}
	return nil, nil
}

// RecvIdentLit 返回 decl.Recv Ident 字面描述. 不含 decl.Name.
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

func FuncParamsLit(decl *ast.FuncDecl) (lit string) {
	return "(" + FieldListLit(decl.Type.Params) + ")"
}

func FuncResultsLit(decl *ast.FuncDecl) (lit string) {
	lit = FieldListLit(decl.Type.Results)
	if lit != "" && (len(decl.Type.Results.List) > 1 ||
		len(decl.Type.Results.List[0].Names) != 0) {
		lit = "(" + lit + ")"
	}
	return
}

// FuncLit 返回 FuncDecl 的字面描述. 不含 decl.Name.
func FuncLit(decl *ast.FuncDecl) (lit string) {
	lit = FuncResultsLit(decl)
	if lit == "" {
		lit = FuncParamsLit(decl)
	} else {
		lit = FuncParamsLit(decl) + " " + lit
	}
	if decl.Name != nil {
		lit = decl.Name.String() + lit
	}
	recv := RecvIdentLit(decl)
	if recv == "" {
		lit = "func " + lit
	} else {
		lit = "func (" + recv + ") " + lit
	}
	return
}

// MethodLit 返回 FuncDecl 的字面描述. 含 decl.Name.
func MethodLit(decl *ast.FuncDecl) (lit string) {
	lit = FuncResultsLit(decl)
	if lit == "" {
		lit = FuncParamsLit(decl)
	} else {
		lit = FuncParamsLit(decl) + " " + lit
	}
	if decl.Name != nil {
		lit = decl.Name.String() + lit
	}

	if decl.Recv != nil && len(decl.Recv.List) != 0 {
		lit = "(" + FieldListLit(decl.Recv) + ") " + lit
	}

	lit = "func " + lit
	return
}

// FieldListLit 返回 ast.FieldList.List 的字面值.
// 该方法仅适用于:
//  ast.FuncDecl.Recv.List
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

func IdentsLit(idents []*ast.Ident) (lit string) {
	if len(idents) == 0 {
		return
	}
	lit = idents[0].String()
	for i := 1; i < len(idents); i++ {
		lit += ", " + idents[i].String()
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
			s += "import (\n\t" + im.Path.Value + nl
		} else {
			s += "\t" + im.Path.Value + nl
		}
	}
	s += ")\n"
	return
}
