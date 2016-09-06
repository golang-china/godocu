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
	trans := SortDecl(source)

	for _, node := range target {
		first := true
		tdecl := node.(*ast.GenDecl)
		for _, tspec := range tdecl.Specs {
			lit = SpecIdentLit(tspec)
			if lit == "_" {
				continue
			}
			spec, decl, _ := trans.SearchSpec(lit)
			if spec == nil {
				continue
			}

			tdoc, tcomm = SpecComment(tspec)
			sdoc, scomm = SpecComment(spec)

			// 解决分组差异
			// 	// target group
			// 	var (
			// 		// target doc
			// 		target = 1
			// 	)
			//	// target doc
			// 	var target = 1
			if first {
				first = false
				if decl.Lparen.IsValid() == tdecl.Lparen.IsValid() {
					MergeDoc(decl.Doc, tdecl.Doc)
					MergeDoc(sdoc, tdoc)
				} else if tdecl.Lparen.IsValid() {
					MergeDoc(decl.Doc, tdoc)
				} else {
					MergeDoc(sdoc, tdecl.Doc)
				}
			} else {
				MergeDoc(sdoc, tdoc)
			}

			// 尾注释
			if scomm != nil && tcomm != nil {
				replaceComment(tcomm, scomm)
			}

			if decl.Tok != token.TYPE || tdecl.Tok != token.TYPE {
				continue
			}

			t, _ := tspec.(*ast.TypeSpec)
			s, _ := spec.(*ast.TypeSpec)
			if s == nil || t == nil {
				continue
			}
			switch tt := t.Type.(type) {
			case *ast.StructType:
				ss, _ := s.Type.(*ast.StructType)
				if ss != nil && tt != nil {
					mergeFieldsDoc(ss.Fields, tt.Fields)
				}
			case *ast.InterfaceType:
				ss, _ := s.Type.(*ast.InterfaceType)
				if ss != nil && tt != nil {
					mergeFieldsDoc(ss.Methods, tt.Methods)
				}
			}
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
			if f == nil {
				continue
			}

			MergeDoc(f.Doc, field.Doc)
			// 尾注释用替换
			replaceComment(field.Comment, f.Comment)

			break
		}
	}
}

// MergeDoc 合并 source.List 到 target.list 底部.
// 插入分隔占位字符串 ___GoDocu_Dividing_line___
func MergeDoc(source, target *ast.CommentGroup) {
	if source != nil && target != nil && !EqualComment(source, target) {
		target.List = append(target.List, comment_Dividing_line)
		target.List = append(target.List, source.List...)
	}
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
		if tdecl != nil {
			MergeDoc(decl.Doc, tdecl.Doc)
		}
	}
	return
}
