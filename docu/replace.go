package docu

import (
	"go/ast"
	"go/token"
	"strings"
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
	var lit string
	var sdoc, tdoc, scomm, tcomm *ast.CommentGroup
	if len(source) == 0 || len(target) == 0 {
		return
	}
	dd := SortDecl(target)

	for _, node := range source {
		decl := node.(*ast.GenDecl)
		first := true
		for _, spec := range decl.Specs {
			lit = SpecIdentLit(spec)
			if lit == "_" {
				continue
			}
			// 必须清理尾注释
			sdoc, scomm = SpecComment(spec)
			ClearComment(src.Comments, scomm)

			tspec, tdecl, _ := dd.SearchSpec(lit)
			if tspec == nil {
				continue
			}

			tdoc, tcomm = SpecComment(tspec)

			// 尾注释
			ClearComment(dst.Comments, tcomm)
			replaceComment(tcomm, scomm)

			// 独立注释
			if sdoc != decl.Doc && tdoc != tdecl.Doc {
				replaceDoc(dst, src, tdoc, sdoc)
			}

			// 分组或者非分组注释
			if first && decl.Lparen.IsValid() == tdecl.Lparen.IsValid() {
				replaceDoc(dst, src, tdecl.Doc, decl.Doc)
			}
			first = false

			if decl.Tok != token.TYPE {
				continue
			}
			// StructType
			stype, _ := spec.(*ast.TypeSpec)
			ttype, _ := tspec.(*ast.TypeSpec)

			st, _ := stype.Type.(*ast.StructType)
			tt, _ := ttype.Type.(*ast.StructType)
			if st == nil || tt == nil {
				continue
			}
			replaceFieldsDoc(dst, src, tt.Fields, st.Fields)
		}
	}
	return
}

func replaceFieldsDoc(dst, src *ast.File, target, source *ast.FieldList) {
	if source == nil || target == nil ||
		len(source.List) == 0 || len(target.List) == 0 {
		return
	}
	for _, field := range target.List {
		if field == nil || field.Doc == nil && field.Comment == nil {
			continue
		}
		for _, ident := range field.Names {
			lit := ident.String()
			if lit == "_" {
				continue
			}
			// 必须清理尾注释
			ClearComment(dst.Comments, field.Comment)
			f, _ := findField(source, lit)
			if f == nil {
				continue
			}
			// 尾注释
			ClearComment(src.Comments, f.Comment)
			replaceComment(field.Comment, f.Comment)
			replaceDoc(dst, src, field.Doc, f.Doc)
			break
		}
	}
}

// replaceComment 替换尾注释
func replaceComment(target, source *ast.CommentGroup) {
	if source == nil || target == nil || EqualComment(source, target) ||
		strings.Index(target.Text(), " // ") != -1 ||
		strings.Index(source.Text(), " // ") == -1 {
		return
	}
	ReplaceDoc(target, source)
}

func replaceDoc(dst, src *ast.File, target, source *ast.CommentGroup) {
	// source 必须是翻译, 且 target 无 origin 才能替换
	// 其实是合并
	if target == nil || OriginDoc(src.Comments, source) == nil {
		return
	}

	if OriginDoc(dst.Comments, target) == nil {
		MergeDoc(source, target)
	}
}

// ReplaceDoc 替换 target.List 为 source.list.
// 保持 target.Pos(), target.End() 不变
func ReplaceDoc(target, source *ast.CommentGroup) {
	pos, end := target.Pos(), target.End()
	if !pos.IsValid() {
		pos = 1 << 30
	}
	if !end.IsValid() {
		end = 1 << 30
	}

	if len(target.List) != 0 {
		target.List = target.List[:0]
	}
	target.List = append(target.List, source.List...)
	cg := target.List[0]
	cg.Slash = pos
	if len(target.List) > 1 {
		cg = target.List[len(target.List)-1]
		cg.Slash = token.Pos(int(end) - len(cg.Text))
	}
}

func replaceFuncDecls(dst, src *ast.File, target, source []ast.Decl) {
	ss := SortDecl(source)
	dd := SortDecl(target)
	if ss.Len() == 0 || dd.Len() == 0 {
		return
	}

	for _, node := range ss {
		decl := node.(*ast.FuncDecl)
		if decl.Doc == nil {
			continue
		}

		tdecl := dd.SearchFunc(FuncIdentLit(decl))
		if tdecl == nil {
			continue
		}

		if tdecl.Doc == nil {
			if OriginDoc(src.Comments, decl.Doc) != nil {
				tdecl.Doc = decl.Doc
			}
			continue
		}
		replaceDoc(dst, src, tdecl.Doc, decl.Doc)
	}
	return
}
