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
	var tdecl *ast.GenDecl

	if len(source) == 0 || len(target) == 0 {
		return
	}
	dd := SortDecl(target)

	var lit string

	for _, node := range source {
		decl := node.(*ast.GenDecl)
		if decl.Tok == token.IMPORT || len(decl.Specs) == 0 {
			continue
		}
		tdecl = nil
		for _, spec := range decl.Specs {
			lit = SpecIdentLit(spec)
			if lit == "_" {
				continue
			}
			if tdecl == nil {
				targ := dd.Search(lit)
				if targ != nil {
					tdecl = targ.(*ast.GenDecl)
				}
			}
			dspec, _, _ := dd.SearchSpec(lit)
			replaceSpec(decl.Tok, dst, src, dspec, spec)
		}

		if tdecl == nil || len(decl.Specs) == 0 || len(tdecl.Specs) == 0 {
			continue
		}
		// 此代码兼容 无分组情况
		replaceDoc(dst, src, tdecl.Doc, decl.Doc)
	}
	return
}

func replaceSpec(tok token.Token, dst, src *ast.File, target, source ast.Spec) {
	if source == nil || target == nil {
		return
	}
	switch tok {
	case token.VAR, token.CONST:
		s, _ := source.(*ast.ValueSpec)
		d, _ := target.(*ast.ValueSpec)

		replaceDoc(dst, src, d.Doc, s.Doc)
	case token.TYPE:
		s, _ := source.(*ast.TypeSpec)
		d, _ := target.(*ast.TypeSpec)
		replaceDoc(dst, src, d.Doc, s.Doc)
	}
}

func replaceDoc(dst, src *ast.File, target, source *ast.CommentGroup) {
	// source 必须要有翻译, 才可能替换
	if equalComment(source, target) || transOrigin(src, source) == nil {
		return
	}
	// target 目标没有翻译, 采用合并
	if transOrigin(dst, target) == nil {
		MergeDoc(target, source)
		target.List = source.List
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
