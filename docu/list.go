package docu

import (
	"go/ast"
	"go/token"
	"strings"
)

// Info 表示单个包文档信息.
type Info struct {
	Import   string // 导入路径
	Synopsis string // 一句话包摘要
	// Readme 该包下 readme 文件名
	Readme   string `json:",omitempty"`
	Progress int    // 翻译完成度
}

// List 表示在同一个 repo 下全部包文档信息.
type List struct {
	// Repo 是原源代码所在托管 git 仓库地址.
	// 如果无法识别值为 "localhost"
	Repo string

	// Description 一句话介绍 Repo 或列表
	// Readme 整个 list 的 readme 文件名
	Description, Readme string `json:",omitempty"`

	// 文档文件名
	Filename string
	// Ext 表示除 "go" 格式文档之外的扩展名.
	// 例如: "md text"
	// 该值由使用者手工设置, Godocu 只是保留它.
	Ext string `json:",omitempty"`

	// Subdir 表示文档文件位于 golist.json 所在目录那个子目录.
	// 该值由使用者手工设置, Godocu 只是保留它.
	Subdir string `json:",omitempty"`

	Package []Info // 所有包的信息
}

// TranslationProgress 返回 file 的翻译完成度.
// 参数 file 应该是单文件的 Godocu 风格翻译文档.
func TranslationProgress(file *ast.File) int {
	var origin, trans int
	doc := file.Doc
	comments := file.Comments

	if doc != nil {
		origin++
		pos := findCommentPrev(doc.Pos(), comments)
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
			if !n.Lparen.IsValid() {
				doc = n.Doc
				break
			}
			for _, spec := range n.Specs {
				if spec == nil {
					continue
				}
				switch n.Tok {
				case token.VAR, token.CONST:
					s, _ := spec.(*ast.ValueSpec)
					doc = s.Doc
				case token.TYPE:
					s, _ := spec.(*ast.TypeSpec)
					doc = s.Doc
				}
				if doc == nil {
					continue
				}
				origin++
				pos := findCommentPrev(doc.Pos(), comments)
				if pos != -1 {
					if !equalComment(doc, comments[pos]) {
						trans++
					}
					comments = comments[pos+1:]
				}
			}
			continue
		case *ast.FuncDecl:
			doc = n.Doc
		default:
			doc = nil
		}

		if doc == nil {
			continue
		}

		origin++
		pos := findCommentPrev(doc.Pos(), comments)
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

// transOrigin 返回 file 中的 trans 的原文, 该算法不严谨.
// file 必须是 GodocuStyle
func transOrigin(file *ast.File, trans *ast.CommentGroup) *ast.CommentGroup {
	if trans == nil || len(trans.List) == 0 {
		return nil
	}

	pos := findCommentPrev(trans.Pos(), file.Comments)
	if pos == -1 {
		return nil
	}
	return file.Comments[pos]
}

// EqualComment 简单比较两个 ast.CommentGroup 值是否一样
func EqualComment(a, b *ast.CommentGroup) bool {
	return equalComment(a, b)
}

func equalComment(a, b *ast.CommentGroup) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
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
	if int(pos) > 0 {
		for i, cg := range comments {
			if cg.End() > pos {
				return -1
			}
			switch pos - cg.End() {
			case 2, 3, 6:
				if len(cg.List) == 0 ||
					strings.HasPrefix(cg.List[0].Text, "// +build") {
					return -1
				}
				return i
			}
		}
	}
	return -1
}
