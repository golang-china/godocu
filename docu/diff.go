package docu

import (
	"go/ast"
	"io"
	"strconv"
	"strings"
)

// FormDiff 对比输出 source, target 的排版或值差异, 返回是否有差异及发生的错误.
func FormDiff(w io.Writer, source, target string) (diff bool, err error) {
	if diff = source != target; diff {
		err = diffOut(lineString(source) == lineString(target), w, source, target)
	}
	return
}

// DiffFormOnly 返回两个字符串是否只是格式不同.
func DiffFormOnly(source, target string) bool {
	return source != target && lineString(source) == lineString(target)
}

// TextDiff 对比输出 source, target 的值差异, 返回是否有差异及发生的错误.
func TextDiff(w io.Writer, source, target string) (diff bool, err error) {
	if diff = source != target; diff {
		err = diffOut(false, w, source, target)
	}
	return
}

func diffOut(form bool, w io.Writer, source, target string) error {
	const prefix = "    "
	if source == "" {
		source = "none"
	}
	if target == "" {
		target = "none"
	}
	if form {
		return fprint(w, "FORM:\n", LineWrapper(source, prefix, 80), "DIFF:\n", LineWrapper(target, prefix, 80), nl)
	}
	return fprint(w, "TEXT:\n", LineWrapper(source, prefix, 80), "DIFF:\n", LineWrapper(target, prefix, 80), nl)
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

// FirstDiff 对比输出两个已排序 ast.File 首个差异, 返回是否有差异及发生的错误.
func FirstDiff(w io.Writer, source, target *ast.File) (diff bool, err error) {
	const nl = "\n\n"
	diff, err = TextDiff(w, "package "+source.Name.String(), "package "+target.Name.String())
	if diff || err != nil {
		return
	}

	diff, err = TextDiff(w, source.Doc.Text(), target.Doc.Text())
	if diff || err != nil {
		return
	}

	diff, err = TextDiff(w, ImportsString(source.Imports), ImportsString(target.Imports))
	if diff || err != nil {
		return
	}

	return firstDecls(w, source.Decls, target.Decls)
}

func firstDecls(w io.Writer, source, target []ast.Decl) (diff bool, err error) {
	sd, so := declsOf(ConstNum, source, 0)
	dd, do := declsOf(ConstNum, target, 0)
	diff, err = diffGenDecls(w, "Const ", sd, dd)
	if diff || err != nil {
		return
	}

	sd, so = declsOf(VarNum, source, so)
	dd, do = declsOf(VarNum, target, do)
	diff, err = diffGenDecls(w, "Var ", sd, dd)
	if diff || err != nil {
		return
	}

	sd, so = declsOf(TypeNum, source, so)
	dd, do = declsOf(TypeNum, target, do)
	diff, err = diffGenDecls(w, "Type ", sd, dd)
	if diff || err != nil {
		return
	}

	sd, so = declsOf(FuncNum, source, so)
	dd, do = declsOf(FuncNum, target, do)
	diff, err = diffFuncDecls(w, "Func ", sd, dd)
	if diff || err != nil {
		return
	}

	sd, so = declsOf(MethodNum, source, so)
	dd, do = declsOf(MethodNum, target, do)
	return diffFuncDecls(w, "Method ", sd, dd)
}

// Diff 对比输出两个已排序 ast.File 差异, 返回是否有差异及发生的错误.
// 如果包名称不同, 停止继续对比.
func Diff(w io.Writer, source, target *ast.File) (diff bool, err error) {
	const nl = "\n\n"
	var out bool
	diff, err = TextDiff(w, "package "+source.Name.String(), "package "+target.Name.String())
	if diff || err != nil {
		return
	}
	out, err = TextDiff(w, source.Doc.Text(), target.Doc.Text())
	if diff = diff || out; err != nil {
		return
	}

	out, err = TextDiff(w, ImportsString(source.Imports), ImportsString(target.Imports))
	if diff = diff || out; err != nil {
		return
	}

	out, err = diffDecls(w, source.Decls, target.Decls)
	diff = diff || out
	return
}

// diffDecls 对比输出两个已排序 []ast.Decl 差异, 返回是否有差异及发生的错误.
// diffDecls 不比较 import 声明
func diffDecls(w io.Writer, source, target []ast.Decl) (diff bool, err error) {
	var out bool
	sd, so := declsOf(ConstNum, source, 0)
	dd, do := declsOf(ConstNum, target, 0)
	out, err = diffGenDecls(w, "Const ", sd, dd)
	if diff = diff || out; err != nil {
		return
	}

	sd, so = declsOf(VarNum, source, so)
	dd, do = declsOf(VarNum, target, do)
	out, err = diffGenDecls(w, "Var ", sd, dd)
	if diff = diff || out; err != nil {
		return
	}

	sd, so = declsOf(TypeNum, source, so)
	dd, do = declsOf(TypeNum, target, do)
	out, err = diffGenDecls(w, "Type ", sd, dd)
	if diff = diff || out; err != nil {
		return
	}

	sd, so = declsOf(FuncNum, source, so)
	dd, do = declsOf(FuncNum, target, do)
	out, err = diffFuncDecls(w, "Func ", sd, dd)
	if diff = diff || out; err != nil {
		return
	}

	sd, so = declsOf(MethodNum, source, so)
	dd, do = declsOf(MethodNum, target, do)
	out, err = diffFuncDecls(w, "Method ", sd, dd)
	diff = diff || out
	return
}

// 需要优化 SortDecl 搜索效率

func diffGenDecls(w io.Writer, prefix string, source, target []ast.Decl) (diff bool, err error) {
	ss := SortDecl(source)
	dd := SortDecl(target)
	if ss.Len() == 0 && dd.Len() == 0 {
		return
	}
	if ss.Len() == 0 || dd.Len() == 0 {
		return TextDiff(w, prefix+strconv.Itoa(ss.Len()), prefix+strconv.Itoa(dd.Len()))
	}
	var lit string

	for _, node := range ss {
		decl := node.(*ast.GenDecl)
		for _, spec := range decl.Specs {
			lit = SpecIdentLit(spec)
			targ := dd.SearchSpec(lit)
			if targ == nil {
				diff, err = true, diffOut(false, w, prefix+lit, "")
				if err != nil {
					return
				}
				continue
			}
			// 类型
			slit, dlit := SpecTypeLit(spec), SpecTypeLit(targ)

			if slit != dlit {
				diff, err = true, diffOut(false, w, prefix+lit+" "+slit, prefix+lit+" "+dlit)
				if err != nil {
					return
				}
				continue
			}
			// 文档
			slit, dlit = SpecDoc(spec), SpecDoc(targ)

			if slit != dlit {
				diff, err = true, diffOut(false, w, prefix+lit+" doc:\n\n"+slit, prefix+lit+" doc:\n\n"+dlit)
			}
			if err != nil {
				return
			}
		}
	}
	// 第二次只对比没有的
	ss, dd = dd, ss
	for _, node := range ss {
		decl := node.(*ast.GenDecl)
		for _, spec := range decl.Specs {
			lit = SpecIdentLit(spec)
			targ := dd.SearchSpec(lit)
			if targ == nil {
				diff, err = true, diffOut(false, w, "", prefix+lit)
				if err != nil {
					return
				}
			}
		}
	}
	return
}

func diffFuncDecls(w io.Writer, prefix string, source, target []ast.Decl) (diff bool, err error) {
	ss := SortDecl(source)
	dd := SortDecl(target)
	if ss.Len() == 0 && dd.Len() == 0 {
		return
	}
	if ss.Len() == 0 || dd.Len() == 0 {
		return TextDiff(w, prefix+strconv.Itoa(ss.Len()), prefix+strconv.Itoa(dd.Len()))
	}
	var lit string
	for _, node := range ss {
		spec := node.(*ast.FuncDecl)
		lit = FuncIdentLit(spec)
		targ := dd.Search(lit)
		if targ == nil {
			diff, err = true, diffOut(false, w, FuncLit(spec), "")
			if err != nil {
				return
			}
			continue
		}
		slit, dlit := FuncLit(spec), FuncLit(targ.(*ast.FuncDecl))
		if slit != dlit {
			diff, err = true, diffOut(false, w, slit, dlit)
			if err != nil {
				return
			}
			continue
		}
		sdoc, ddoc := spec.Doc.Text(), targ.(*ast.FuncDecl).Doc.Text()
		if sdoc != ddoc {
			diff, err = true, diffOut(false, w, slit+"\n\n"+sdoc, dlit+"\n\n"+ddoc)
			if err != nil {
				return
			}
		}
	}
	ss, dd = dd, ss
	for _, node := range ss {
		spec := node.(*ast.FuncDecl)
		lit = FuncIdentLit(spec)
		targ := dd.Search(lit)
		if targ == nil {
			diff, err = true, diffOut(false, w, "", FuncLit(spec))
			if err != nil {
				return
			}
		}
	}
	return
}

func declsOf(num int, decls []ast.Decl, offset int) ([]ast.Decl, int) {
	first, last := -1, len(decls)
	if offset >= 0 && offset < len(decls) {
		for i, node := range decls[offset:] {
			if first == -1 {
				if NodeNumber(node) == num {
					first = offset + i
				}
			} else if NodeNumber(node) > num {
				last = offset + i
				break
			}
		}
	}
	if first == -1 {
		return nil, offset
	}
	return decls[first:last], last
}
