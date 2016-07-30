package docu

import (
	"go/ast"
	"go/token"
)

// do not change this
var docu_Dividing_line = []*ast.Comment{
	&ast.Comment{Text: "//"},
	&ast.Comment{Text: "// ___GoDocu_Dividing_line___"},
	&ast.Comment{Text: "//"},
}

// MergeDecls 插入(合并) source,target 中相匹配的 Ident 的文档到 target 注释顶部.
// 细节:
//    忽略 ImportSpec
//    只是排版不同不会被合并
//    插入分隔占位字符串 ___GoDocu_Dividing_line___
func MergeDecls(source, target []ast.Decl) {
	sd, so := declsOf(ConstNum, source, 0)
	dd, do := declsOf(ConstNum, target, 0)
	mergeGenDecls(sd, dd)

	sd, so = declsOf(VarNum, source, so)
	dd, do = declsOf(VarNum, target, do)
	mergeGenDecls(sd, dd)

	sd, so = declsOf(TypeNum, source, so)
	dd, do = declsOf(TypeNum, target, do)
	mergeGenDecls(sd, dd)

	sd, so = declsOf(FuncNum, source, so)
	dd, do = declsOf(FuncNum, target, do)
	mergeFuncDecls(sd, dd)

	sd, so = declsOf(MethodNum, source, so)
	dd, do = declsOf(MethodNum, target, do)
	mergeFuncDecls(sd, dd)
	return
}

// 需要优化 SortDecl 搜索效率

// mergeGenDecls 负责 ValueSpec, TypeSpec
func mergeGenDecls(source, target []ast.Decl) {
	ss := SortDecl(source)
	dd := SortDecl(target)
	if ss.Len() == 0 || dd.Len() == 0 {
		return
	}

	var lit string
	for _, node := range ss {
		decl := node.(*ast.GenDecl)
		if decl.Tok == token.IMPORT ||
			decl.Doc == nil || len(decl.Doc.List) == 0 {
			continue
		}

		lit = DeclIdentLit(decl)
		targ := dd.Search(lit)
		if targ == nil {
			continue
		}
		tdecl, ok := targ.(*ast.GenDecl)
		if !ok || len(decl.Specs) == 0 || len(tdecl.Specs) == 0 {
			continue
		}
		if tdecl.Doc == nil || len(tdecl.Doc.List) == 0 {
			tdecl.Doc = decl.Doc
			continue
		}

		sdoc, ddoc := decl.Doc.Text(), tdecl.Doc.Text()
		if sdoc == ddoc || lineString(sdoc) == lineString(ddoc) {
			continue
		}
		MergeDoc(decl.Doc, tdecl.Doc)
	}
	return
}

// MergeDoc 合并 source.List 到 target.list 顶部.
func MergeDoc(source, target *ast.CommentGroup) {
	list := target.List
	target.List = nil
	target.List = append(target.List, source.List...)
	target.List = append(target.List, docu_Dividing_line...)
	target.List = append(target.List, list...)
}

func mergeFuncDecls(source, target []ast.Decl) {
	ss := SortDecl(source)
	dd := SortDecl(target)
	if ss.Len() == 0 || dd.Len() == 0 {
		return
	}

	var lit string
	for _, node := range ss {
		decl := node.(*ast.FuncDecl)
		if decl.Doc == nil || len(decl.Doc.List) == 0 {
			continue
		}

		lit = FuncIdentLit(decl)
		targ := dd.Search(lit)
		if targ == nil {
			continue
		}
		tdecl, ok := targ.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if tdecl.Doc == nil || len(tdecl.Doc.List) == 0 {
			tdecl.Doc = decl.Doc
			continue
		}

		sdoc, ddoc := decl.Doc.Text(), tdecl.Doc.Text()
		if sdoc == ddoc || lineString(sdoc) == lineString(ddoc) {
			continue
		}
		MergeDoc(decl.Doc, tdecl.Doc)
	}
	return
}
