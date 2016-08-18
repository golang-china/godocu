package docu

import (
	"go/ast"
	"go/token"
)

// Replace 用 source 翻译文档替换 target 中相匹配的 Ident 的翻译文档.
// 细节:
//    target, source 必须是双语翻译文档
//    忽略 ImportSpec
//    替换后 target 中的文档 Text() 改变, Pos(), End() 不变.
func Replace(target, source *ast.File) {

	if !IsGodocuFile(source) || !IsGodocuFile(target) {
		return
	}
	replaceDoc(target, source, target.Doc, source.Doc)

	sd, so := declsOf(ConstNum, source.Decls, 0)
	dd, do := declsOf(ConstNum, target.Decls, 0)
	replaceGenDecls(target, source, dd, sd)

	sd, so = declsOf(VarNum, source.Decls, so)
	dd, do = declsOf(VarNum, target.Decls, do)
	replaceGenDecls(target, source, dd, sd)

	sd, so = declsOf(TypeNum, source.Decls, so)
	dd, do = declsOf(TypeNum, target.Decls, do)
	replaceGenDecls(target, source, dd, sd)

	sd, so = declsOf(FuncNum, source.Decls, so)
	dd, do = declsOf(FuncNum, target.Decls, do)
	replaceFuncDecls(target, source, dd, sd)

	sd, so = declsOf(MethodNum, source.Decls, so)
	dd, do = declsOf(MethodNum, target.Decls, do)
	replaceFuncDecls(target, source, dd, sd)
	return
}

// 需要优化 SortDecl 搜索效率

// replaceGenDecls 负责 ValueSpec, TypeSpec
func replaceGenDecls(dst, src *ast.File, target, source []ast.Decl) {
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
		replaceDoc(dst, src, tdecl.Doc, decl.Doc)
	}
	return
}

func replaceDoc(dst, src *ast.File, target, source *ast.CommentGroup) {
	var so, do *ast.CommentGroup

	so = transOrigin(src, source)
	if so == nil || equalComment(so, source) {
		return
	}

	do = transOrigin(dst, target)
	if do == nil || !equalComment(do, target) {
		return
	}
	ReplaceDoc(target, source)
}

// ReplaceDoc 替换 target.List 为 source.list.
// 保持 target.Pos(), target.End() 不变
func ReplaceDoc(target, source *ast.CommentGroup) {
	pos, end := target.Pos(), target.End()
	if target.List != nil {
		target.List = target.List[:0]
	}
	target.List = append(target.List, source.List...)
	cg := target.List[0]
	cg.Slash = pos
	cg = target.List[len(target.List)-1]
	cg.Slash = token.Pos(int(end) - len(cg.Text))
}

func replaceFuncDecls(dst, src *ast.File, target, source []ast.Decl) {
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

		replaceDoc(dst, src, tdecl.Doc, decl.Doc)
	}
	return
}
