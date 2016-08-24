package docu

import (
	"go/ast"
	"go/token"
)

const GoDocu_Dividing_line = "___GoDocu_Dividing_line___"

const indent_GoDocu_Dividing_line = "    //" + GoDocu_Dividing_line

// do not change this
var comment_Dividing_line = &ast.Comment{Text: "//___GoDocu_Dividing_line___"}

// MergeDeclsDoc 添加 source 与 target 中匹配的标识符文档到 target 注释底部
// 细节:
//    忽略 ImportSpec
//    只是排版不同不会被合并
//    target 应该具有良好的结构, 比如来自源代码
//    用 source 中的尾注释替换 target 中的尾注释
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
			tspec, tdecl, _ := dd.SearchSpec(lit)
			if tspec == nil {
				continue
			}

			tdoc, tcomm = SpecComment(tspec)
			sdoc, scomm = SpecComment(spec)

			// 尾注释
			if scomm != nil && tcomm != nil {
				ReplaceDoc(tcomm, scomm)
			}

			// 独立注释
			if sdoc != nil && tdoc != nil && sdoc != decl.Doc && tdoc != tdecl.Doc &&
				!equalComment(sdoc, tdoc) {

				MergeDoc(sdoc, tdoc)
			}

			// 分组注释
			if first && decl.Lparen.IsValid() && tdecl.Lparen.IsValid() &&
				decl.Doc != nil && tdecl.Doc != nil && !equalComment(decl.Doc, tdecl.Doc) {

				MergeDoc(decl.Doc, tdecl.Doc)
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
			mergeFieldsDoc(st.Fields, tt.Fields)
		}
	}
	return
}

// mergeFieldsDoc 保持 target 的结构和尾注释
func mergeFieldsDoc(source, target *ast.FieldList) {
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
			f, _ := findField(source, lit)
			if f != nil && f.Doc != nil && field.Doc != nil &&
				!equalComment(f.Doc, field.Doc) {

				MergeDoc(f.Doc, field.Doc)
			}
			// 尾注释
			if f != nil && f.Comment != nil && field.Comment != nil &&
				!equalComment(f.Comment, field.Comment) {
				ReplaceDoc(field.Comment, f.Comment)
			}

			break
		}
	}
}

// MergeDoc 合并 source.List 到 target.list 底部.
// 保持 target.Pos(), target.End() 不变
// 插入分隔占位字符串 ___GoDocu_Dividing_line___
func MergeDoc(source, target *ast.CommentGroup) {
	end := target.End()
	target.List = append(target.List, comment_Dividing_line)
	target.List = append(target.List, source.List...)
	cg := target.List[len(target.List)-1]
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
		if decl == nil || decl.Doc == nil {
			continue
		}

		lit = FuncIdentLit(decl)
		tdecl := dd.SearchFunc(lit)
		if tdecl == nil || tdecl.Doc == nil || equalComment(decl.Doc, tdecl.Doc) {
			continue
		}
		MergeDoc(decl.Doc, tdecl.Doc)
	}
	return
}
