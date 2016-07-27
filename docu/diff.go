package docu

import (
	"go/ast"
	"io"
	"strings"
)

func sameForm(w io.Writer, title, new, old string) bool {
	if new == old {
		return true
	}
	if lineString(new) == lineString(old) {
		fprint(w, "[FORM] ", title)
	} else {
		fprint(w, "[TEXT] ", title)
	}
	return false
}

func diffText(w io.Writer, title, new, old string) bool {
	if new == old || lineString(new) == lineString(old) {
		return true
	}
	fprint(w, "[TEXT] ", title)
	return false
}

// lineString 对 str 进行单行合并, 剔除空白行
func lineString(str string) string {
	s := strings.Split(str, "\n")
	str = ""
	for i := 0; i < len(s); i++ {
		v := strings.TrimSpace(s[i])
		if v != "" {
			str += v
		}
	}
	return str
}

// Same 简单比较并返回两个已排序的 ast.File 是否相同.
// 遇到任何不同就停止比较, 并数据简单信息.
func Same(w io.Writer, new, old *ast.File) (same bool) {
	same = sameForm(w, "package name", new.Name.String(), old.Name.String()) &&
		sameForm(w, "package doc", new.Doc.Text(), old.Doc.Text())

	if !same {
		return
	}

	if same && (len(new.Imports) != len(old.Imports) ||
		ImportsString(new.Imports) != ImportsString(old.Imports)) {

		fprint(w, "[TEXT] imports")
		return false
	}

	if len(new.Decls) != len(old.Decls) {
		fprint(w, "[TEXT] Decls")
		return false
	}
	nd, od := new.Decls, old.Decls
	count := len(nd)
	for i := 0; i < count; i++ {
		num := NodeNumber(nd[i])
		if same = num == NodeNumber(od[i]); !same {
			fprint(w, "[TEXT] Type")
			return
		}
		if num == ImportNum {
			continue
		}
		if num == FuncNum || num == MethodNum {
			a := nd[i].(*ast.FuncDecl)
			b := od[i].(*ast.FuncDecl)
			n, o := FuncLit(a), FuncLit(b)
			if same = n == o; !same {
				fprint(w, "[TEXT] ", n, " <> ", o)
				return
			}
			if same = sameForm(w, "doc "+n, a.Doc.Text(), b.Doc.Text()); !same {
				return
			}
			continue
		}
		a := nd[i].(*ast.GenDecl)
		b := nd[i].(*ast.GenDecl)
		if a == nil || b == nil {
			continue
		}
		if same = len(a.Specs) == len(b.Specs); !same {
			fprint(w, "[TEXT] ", a.Tok.String(), " length")
			return
		}
		if same = sameForm(w, a.Tok.String(), a.Doc.Text(), b.Doc.Text()); !same {
			return
		}
		c := len(a.Specs)
		for i := 0; i < c; i++ {
			n := SpecIdentLit(a.Specs[i])
			o := SpecIdentLit(b.Specs[i])
			if same = n == o; !same {
				fprint(w, "[TEXT] ", a.Tok.String(), " ", n, " <> ", o)
				return
			}
		}
	}
	return
}
