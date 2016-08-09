package docu

import (
	"go/ast"
	"go/token"
)

// Info 表示单个包文档信息.
type Info struct {
	Import   string // 导入路径
	Synopsis string // 一句话包摘要
	Progress int    // 翻译完成度
	Prefix   string // 例如 "doc" 或 "doc,main,test"
}

// List 表示在同一个 repo 下全部包文档信息
type List struct {
	Repo        string // 托管 git 仓库地址.
	Description string // 一句话介绍 Repo 或列表
	Subdir      string // 文档所在 repo 下的子目录
	Lang        string // 同一个列表具有相同的 Lang
	Markdown    bool   // 是否具有完整的 Markdown 文档
	Info        []Info
}

// TranslationProgress 返回 file 的翻译完成度.
// 参数 file 应该是单文件的 Godocu 风格翻译文档.
func TranslationProgress(file *ast.File) int {
	var origin, trans int
	doc := file.Doc
	comments := file.Comments

	if doc != nil {
		origin++
		pos := findCommentPrev(doc.Pos()-2, comments)
		if pos != -1 {
			if !equalComment(doc, comments[pos]) {
				trans++
			}
			comments = comments[pos+1:]
		}
	}

	for _, node := range file.Decls {
		switch n := node.(type) {
		case *ast.GenDecl:
			if n.Tok == token.IMPORT {
				continue
			}
			doc = n.Doc
		case *ast.FuncDecl:
			doc = n.Doc
		default:
			doc = nil
		}

		if doc == nil {
			continue
		}

		origin++
		pos := findCommentPrev(doc.Pos()-2, comments)
		if pos != -1 {
			if !equalComment(doc, comments[pos]) {
				trans++
			}
			comments = comments[pos+1:]
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

func equalComment(a, b *ast.CommentGroup) bool {
	if len(a.List) != len(b.List) {
		return false
	}
	for i, c := range a.List {
		if c.Text != b.List[i].Text {
			return false
		}
	}
	return true
}
func findCommentPrev(pos token.Pos, comments []*ast.CommentGroup) int {
	for i, cg := range comments {
		if cg.End() == pos {
			return i
		}
	}
	return -1
}
