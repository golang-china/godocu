package docu

import (
	"go/ast"
	"io"
	"strconv"
	"strings"
)

func SameForm(w io.Writer, source, target string) (same bool, err error) {
	return sameForm(w, source, target)
}

func SameText(w io.Writer, source, target string) (same bool, err error) {
	return sameText(w, source, target)
}

func sameForm(w io.Writer, source, target string) (same bool, err error) {
	const prefix = "    "
	if same = source == target; same {
		return
	}
	if lineString(source) == lineString(target) {
		err = fprint(w, "FORM:\n", LineWrapper(source, prefix, 80), "\nDIFF:\n", LineWrapper(target, prefix, 80), nl)
	} else {
		err = fprint(w, "TEXT:\n", LineWrapper(source, prefix, 80), "\nDIFF:\n", LineWrapper(target, prefix, 80), nl)
	}
	return
}

func sameText(w io.Writer, source, target string) (same bool, err error) {
	const prefix = "    "
	if same = source == target; same {
		return
	}
	err = fprint(w, "TEXT:\n", LineWrapper(source, prefix, 80), "\nDIFF:\n", LineWrapper(target, prefix, 80), nl)
	return
}

// lineString 对 str 进行单行合并, 剔除空白行
func lineString(str string) string {
	s := strings.Split(str, "\n")
	str = ""
	for i := 0; i < len(s); i++ {
		str += " " + strings.TrimSpace(s[i])
	}
	return str
}

// Same 返回两个已排序 ast.File 是否相同, 并输出首个差异.
func Same(w io.Writer, source, target *ast.File) (same bool, err error) {
	const nl = "\n\n"
	same, err = sameText(w, "package "+source.Name.String(), "package "+target.Name.String())
	if !same || err != nil {
		return
	}

	same, err = sameForm(w, source.Doc.Text(), target.Doc.Text())
	if !same || err != nil {
		return
	}

	same, err = sameForm(w, ImportsString(source.Imports), ImportsString(target.Imports))
	if !same || err != nil {
		return
	}

	sd, td := source.Decls, target.Decls
	count, total := len(sd), len(td)
	same, err = sameText(w, "Decls length "+strconv.Itoa(count), "Decls length "+strconv.Itoa(total))
	if !same || err != nil {
		return
	}

	for i := 0; i < count; i++ {
		num := NodeNumber(sd[i])
		same, err = sameText(w, numNames[num], numNames[NodeNumber(td[i])])
		if !same || err != nil {
			return
		}

		if num == ImportNum {
			continue
		}

		if num == FuncNum || num == MethodNum {
			a := sd[i].(*ast.FuncDecl)
			b := td[i].(*ast.FuncDecl)
			same, err = sameText(w, FuncLit(a), FuncLit(b))
			if same && err == nil {
				same, err = sameForm(w, a.Doc.Text(), b.Doc.Text())
			}
			if !same || err != nil {
				return
			}
			continue
		}
		a := sd[i].(*ast.GenDecl)
		b := td[i].(*ast.GenDecl)
		if a == nil || b == nil {
			continue
		}
		same, err = sameText(w, "Specs length "+strconv.Itoa(len(a.Specs)), "Specs length "+strconv.Itoa(len(b.Specs)))
		if same && err == nil {
			same, err = sameForm(w, a.Doc.Text(), b.Doc.Text())
		}
		if !same || err != nil {
			return
		}

		c := len(a.Specs)
		for i := 0; i < c; i++ {
			same, err = sameText(w, SpecIdentLit(a.Specs[i]), SpecIdentLit(b.Specs[i]))
			if !same || err != nil {
				return
			}
		}
	}
	return
}
