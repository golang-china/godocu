package docu

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/printer"
	"go/token"
	"io"
	"strings"
	"unicode"

	"golang.org/x/text/width"
)

const nl = "\n"

// Resultsify 返回 " (results)" , 如果 results 含有空格的话.
func Resultsify(results string) string {
	if results == "" {
		return results
	}
	if strings.IndexByte(results, ' ') == -1 {
		return " " + results
	}
	return " (" + results + ")"
}

// ToText 以 4 空格缩进向 go/doc.ToText 输出注释文本.
func ToText(output io.Writer, text string) {
	var buf bytes.Buffer
	if text != "" {
		doc.ToText(&buf, text, "    ", "", 76)
		io.WriteString(output, LineWrapper(buf.String(), "    ", 76))
	}
}

// LineWrapper 在换行中要保留标点符号
var KeepPunct = `,.:;?，．：；？。`

// LineWrapper 把 text 非缩进行超过显示长度 limit 的行插入换行符 "\n".
// 细节:
//	text 行间 tab 按 4 字节宽度计算.
//	preIndent 为每行固定缩进字符串.
//	limit 的长度不包括 preIndent 的长度.
//	位于换行处的标点符被保留.
func LineWrapper(text string, preIndent string, limit int) (wrap string) {
	const nl = "\n"
	var lf, r, next rune
	var isIndent bool
	var last, word string // 最后一行前部和尾部单词

	n, w := 0, 0
	for _, r = range text {
		// 预读取一个
		r, next = next, r
		if r == 0 {
			continue
		}
		switch r {
		case '\r':
			wrap += last + word + nl
			w, last, word = 0, preIndent, ""
			isIndent, lf = false, r
			continue
		case '\n':
			if lf != '\r' {
				wrap += last + word + nl
				w, last, word = 0, preIndent, ""
			}
			lf, isIndent = r, false
			continue
		case '\t':
			// 中间的 tab 要保持, 虽然不大可能产生
			w, last, word = w/4*4+4, last+"\t", ""
			isIndent, lf = isIndent || lf == '\r' || lf == '\n', r
			continue
		}

		isIndent = isIndent || r == ' ' && (lf == '\r' || lf == '\n')
		n, lf = 1, r

		if isIndent {
			wrap += string(r)
			continue
		}

		if r > unicode.MaxLatin1 {
			switch width.LookupRune(r).Kind() {
			case width.EastAsianAmbiguous, width.EastAsianWide, width.EastAsianFullwidth:
				n = 2
			}
		}
		//fmt.Println("...", w, n, string(r))
		w += n
		keep := strings.IndexRune(KeepPunct, next) != -1
		// 多字节及时换行
		if n != 1 {
			if keep || w < limit {
				last += word + string(r)
			} else if w == limit {
				wrap += last + word + string(r) + nl
				w, last = 0, preIndent
			} else {
				wrap += last + word + nl
				w, last = 0, preIndent+string(r)
			}
			word = ""
			continue
		}

		// 保持单词完整性.
		if !keep && r != ' ' {
			word += string(r)
			continue
		}

		if keep || w < limit {
			last, word = last+word+string(r), ""
		} else if w == limit || r == ' ' || r == '　' {
			wrap += last + word + string(r) + nl
			w, last, word = 0, preIndent, ""
		} else {
			wrap += last + nl
			w, last, word = n, preIndent, word+string(r)
		}
	}
	if word != "" {
		wrap += last + word
	}
	if next != 0 {
		wrap += string(next)
	}
	return
}

var config = printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}

// Godoc 仿 godoc 风格向 output 输出 ast.Package.
func Godoc(output io.Writer, paths string, fset *token.FileSet, pkg *ast.Package) (err error) {
	file := ast.MergePackageFiles(pkg,
		ast.FilterFuncDuplicates|ast.FilterUnassociatedComments|ast.FilterImportDuplicates,
	)
	Index(file)
	fmt.Fprintln(output, "PACKAGE DOCUMENTATION\n\npackage", file.Name.String())
	ToText(output, `import "`+paths+`"`)
	if file.Doc != nil {
		fmt.Fprintln(output)
		ToText(output, file.Doc.Text())
	}

	if len(file.Imports) != 0 {
		fmt.Fprint(output, "\nIMPORTS\n\n")
		if len(file.Imports) == 1 {
			fmt.Fprint(output, "import ", file.Name.String(), nl)
		} else {
			fmt.Fprint(output, "import (\n")
			for i, im := range file.Imports {
				if i == 0 {
					fmt.Fprint(output, "    ")
				} else {
					fmt.Fprint(output, "\n    ")
				}
				fmt.Fprint(output, im.Path.Value)
			}
			fmt.Fprint(output, "\n)\n")
		}
	}

	step := ImportNum
	for _, decl := range file.Decls {
		num := NodeNumber(decl)
		if num == ImportNum {
			continue
		}

		if num == FuncNum || num == MethodNum {
			fdecl := decl.(*ast.FuncDecl)
			if !ast.IsExported(fdecl.Name.String()) {
				continue
			}
			if step != num {
				if num == FuncNum {
					fmt.Fprint(output, "\nFUNCTIONS\n")
				} else {
					fmt.Fprint(output, "\nMETHODS\n")
				}
				step = num
			}
			if num == FuncNum {
				fmt.Fprint(output, "\nfunc ")
			} else {
				fmt.Fprint(output, "\nfunc (", RecvLit(fdecl), ") ")
			}
			fmt.Fprint(output, fdecl.Name.String(), "(", FieldListLit(fdecl.Type.Params), ")")
			fmt.Fprint(output, Resultsify(FieldListLit(fdecl.Type.Results)), nl)
			if fdecl.Doc != nil {
				ToText(output, fdecl.Doc.Text())
			}
			continue
		}
		if step != num {
			step = num
			switch num {
			case TypeNum:
				fmt.Fprint(output, "\nTYPES\n")
			case ConstNum:
				fmt.Fprint(output, "\nCONSTANTS\n")
			case VarNum:
				fmt.Fprint(output, "\nVARIABLES\n")
			}
		}
		genDecl := decl.(*ast.GenDecl)
		docGroup := genDecl.Doc
		genDecl.Doc = nil
		fmt.Fprint(output, nl)
		config.Fprint(output, fset, decl)
		fmt.Fprint(output, nl)
		ToText(output, docGroup.Text())
		genDecl.Doc = docGroup
	}
	return nil
}
