package docu

import (
	"go/ast"
	"go/token"
	"strings"
)

// TranslationProgress 返回 file 的翻译完成度.
// 参数 file 应该是单文件的 Godocu 风格翻译文档.
func TranslationProgress(file *ast.File) int {
	var origin, trans int
	comments := file.Comments
	if _, pos := License(file); pos != -1 {
		comments = comments[pos+1:]
	}

	count := func(doc *ast.CommentGroup) {
		if doc == nil {
			return
		}
		origin++
		pos, src := docPosAndOrigin(comments, doc)
		if src != nil && !EqualComment(doc, src) {
			trans++
		}
		comments = comments[pos+1:]
	}
	count(file.Doc)

	for _, node := range file.Decls {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				continue
			}
			count(n.Doc)
			for _, spec := range n.Specs {
				if spec == nil {
					continue
				}

				switch n.Tok {
				case token.VAR, token.CONST:
					s, _ := spec.(*ast.ValueSpec)
					count(s.Doc)
					ClearComment(comments, s.Comment)
				case token.TYPE:
					s, _ := spec.(*ast.TypeSpec)
					count(s.Doc)
					ClearComment(comments, s.Comment)
					st, ok := s.Type.(*ast.StructType)
					if ok && st.Fields != nil {
						for _, n := range st.Fields.List {
							count(n.Doc)
							ClearComment(comments, n.Comment)
						}
					}
				}
			}
			continue
		case *ast.FuncDecl:
			count(n.Doc)
		}
		if len(comments) == 0 {
			break
		}
	}

	if origin == 0 {
		return 100
	}
	return trans * 100 / origin
}

// License 返回 file 中以 copyright 开头的注释,和该注释的偏移量, 如果有的话.
func License(file *ast.File) (lic string, pos int) {
	end := file.Name.Pos()
	if file.Doc != nil {
		end = file.Doc.Pos()
	}
	for i, comm := range file.Comments {
		if comm == nil {
			continue
		}
		if comm.Pos() >= end {
			break
		}
		lic = comm.Text()
		pos = strings.IndexByte(lic, ' ')
		if pos != -1 && "copyright" == strings.ToLower(lic[:pos]) {
			return lic, i
		}
	}
	return "", -1
}

// clearFile 清理 file.Comments 中 package 声明之前的注释. 比如 +build 注释.
func clearFile(file *ast.File) {
	end := file.Name.Pos()
	if file.Doc != nil {
		end = file.Doc.Pos()
	}
	comments := file.Comments
	for i := 0; i < len(comments); i++ {
		if comments[i].Pos() >= end {
			break
		}
		text := comments[i].Text()
		pos := strings.IndexByte(text, ' ')
		if pos == -1 {
			continue
		}
		if text[:pos] == "+build" {
			comments[i] = nil
		}
	}
}

// OriginDoc 返回 file 中的 trans 的原文档.
// 要求 comments 为 GodocuStyle 双语文档, 需要配合 ClearComments, ClearComment.
func OriginDoc(comments []*ast.CommentGroup, trans *ast.CommentGroup) (origin *ast.CommentGroup) {
	_, origin = docPosAndOrigin(comments, trans)
	return
}

func docPosAndOrigin(comments []*ast.CommentGroup, trans *ast.CommentGroup) (int, *ast.CommentGroup) {
	if trans == nil {
		return -1, nil
	}
	pos := indexOf(comments, trans.Pos())
	if pos > 0 && isOrigin(comments[pos-1], trans.Pos()) {
		return pos, comments[pos-1]
	}
	return pos, nil
}

// ClearComment 设置 comments 中的 comment 元素为 nil.
// 现实中清除尾注释, 以便 OriginDoc 能正确计算出原文档.
func ClearComment(comments []*ast.CommentGroup, comment *ast.CommentGroup) {
	if len(comments) == 0 || comment == nil || len(comment.List) == 0 {
		return
	}
	i := indexOf(comments, comment.Pos())
	if i != -1 {
		comments[i] = nil
	}
}

// ClearComments 置 file.Comments 中所有的尾注释元素为 nil.
// 现实中清除尾注释, 以便 OriginDoc 能正确计算出原文档.
func ClearComments(file *ast.File) {
	comments := file.Comments
	for _, decl := range file.Decls {
		switch n := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range n.Specs {
				switch n := spec.(type) {
				case *ast.ValueSpec:
					ClearComment(comments, n.Comment)
				case *ast.ImportSpec:
					ClearComment(comments, n.Comment)
				case *ast.TypeSpec:
					ClearComment(comments, n.Comment)
					st, ok := n.Type.(*ast.StructType)
					if ok && st.Fields != nil {
						for _, n := range st.Fields.List {
							ClearComment(comments, n.Comment)
						}
					}
				}
			}
		}
	}
}

// EqualComment 简单比较两个 ast.CommentGroup 值是否一样
func EqualComment(a, b *ast.CommentGroup) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return len(a.List) == len(b.List) && a.Text() == b.Text()
}

func indexOf(comments []*ast.CommentGroup, pos token.Pos) int {
	for i, cg := range comments {
		if cg == nil || cg.Pos() < pos {
			continue
		}
		if cg.Pos() == pos {
			return i
		}
		break
	}
	return -1
}

func isOrigin(comment *ast.CommentGroup, trans token.Pos) bool {
	if comment == nil {
		return false
	}
	trans -= comment.End()
	return trans == 2 || trans == 3 || trans == 4
}
