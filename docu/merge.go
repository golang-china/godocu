package docu

import (
	"go/ast"
	"go/token"
)

const GoDocu_Dividing_line = "___GoDocu_Dividing_line___"

// do not change this
var comment_Dividing_line = &ast.Comment{Text: "//___GoDocu_Dividing_line___"}

// MergeDeclsDoc 插入(合并) source,target 中相匹配的 Ident 的文档到 target 注释顶部.
// 细节:
//    忽略 ImportSpec
//    只是排版不同不会被合并
//    插入分隔占位字符串 ___GoDocu_Dividing_line___
func MergeDeclsDoc(source, target []ast.Decl) {
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
	var targ ast.Decl

	if len(source) == 0 || len(target) == 0 {
		return
	}
	dd := SortDecl(target)

	var lit string

	for _, node := range source {
		decl := node.(*ast.GenDecl)
		if decl.Tok == token.IMPORT ||
			decl.Doc == nil || len(decl.Doc.List) == 0 {
			continue
		}
		targ = nil
		for _, spec := range decl.Specs {
			lit = SpecIdentLit(spec)
			mergeSpec(decl.Tok, spec, dd.SearchSpec(lit))
			if targ == nil {
				targ = dd.Search(lit)
			}
		}

		if targ == nil {
			continue
		}

		tdecl, _ := targ.(*ast.GenDecl)
		if tdecl == nil {
			continue
		}

		if tdecl.Doc == nil || len(tdecl.Doc.List) == 0 {
			tdecl.Doc = decl.Doc
			continue
		}
		if !equalComment(decl.Doc, tdecl.Doc) {
			MergeDoc(decl.Doc, tdecl.Doc)
		}
	}
	return
}

func mergeSpec(tok token.Token, source, target ast.Spec) {
	if source == nil || target == nil {
		return
	}
	switch tok {
	case token.VAR, token.CONST:
		src, _ := source.(*ast.ValueSpec)
		dst, _ := target.(*ast.ValueSpec)
		if src == nil || dst == nil ||
			src.Doc == nil || dst.Doc == nil ||
			equalComment(src.Doc, dst.Doc) {
			return
		}
		MergeDoc(src.Doc, dst.Doc)
	case token.TYPE:
		src, _ := source.(*ast.TypeSpec)
		dst, _ := target.(*ast.TypeSpec)
		if src == nil || dst == nil ||
			src.Doc == nil || dst.Doc == nil ||
			equalComment(src.Doc, dst.Doc) {
			return
		}
		MergeDoc(src.Doc, dst.Doc)
	}
}

// MergeDoc 合并 source.List 到 target.list 顶部.
// 保持 target.Pos(), target.End() 不变
func MergeDoc(source, target *ast.CommentGroup) {
	pos, end := target.Pos(), target.End()
	list := target.List
	target.List = make([]*ast.Comment, 0, len(source.List)+len(list)+1)
	target.List = append(target.List, source.List...)
	target.List = append(target.List, comment_Dividing_line)
	target.List = append(target.List, list...)
	cg := target.List[0]
	cg.Slash = pos
	cg = target.List[len(target.List)-1]
	cg.Slash = token.Pos(int(end) - len(cg.Text))
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
