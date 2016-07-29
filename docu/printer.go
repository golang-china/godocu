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

// ToText 以 4 空格缩进输出 godoc 风格的纯文本注释.
func ToText(output io.Writer, text string) error {
	return FormatComments(output, text, "    ", 76)
}

// ToSource 以 4 空格缩进输出 go source 风格的注释.
func ToSource(output io.Writer, text string) error {
	return FormatComments(output, text, "// ", 77)
}

// FormatComments 调用 LineWrapper 换行格式化注释 text 输出到 output.
func FormatComments(output io.Writer, text, prefix string, limit int) (err error) {
	var buf bytes.Buffer
	if text != "" {
		// 利用 ToText 的 preIndent 功能
		doc.ToText(&buf, text, "", "    ", 1<<32)
		_, err = io.WriteString(output, LineWrapper(buf.String(), prefix, limit))
	}
	return
}

// KeepPunct 是 LineWrapper 进行换行时行尾要保留标点符号
var KeepPunct = `,.:;?，．：；？。`

// LineWrapper 把 text 非缩进行超过显示长度 limit 的行插入换行符 "\n".
// 细节:
//	text   行间 tab 按 4 字节宽度计算.
//	prefix 为每行前缀字符串.
//	limit  的长度不包括 prefix 的长度.
//	位于换行处的标点符被保留.
func LineWrapper(text string, prefix string, limit int) (wrap string) {
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
			wrap += strings.TrimRight(prefix+last+word, " ") + nl
			w, last, word = 0, "", ""
			isIndent, lf = false, r
			continue
		case '\n':
			if lf != '\r' {
				wrap += strings.TrimRight(prefix+last+word, " ") + nl
				w, last, word = 0, "", ""
			}
			lf, isIndent = r, false
			continue
		case '\t':
			// tab 缩进替换为 4 空格, 保持行间 tab
			if lf == '\n' || lf == '\r' {
				w, last, word = w/4*4+4, last+"    ", ""
				isIndent = true
			} else {
				w, last, word = w/4*4+4, last+"\t", ""
			}
			continue
		case ' ':
			// 行首连续两个空格算做缩进
			if next == ' ' && (lf == '\n' || lf == '\r') {
				isIndent = true
			}
		}
		n, lf = 1, r

		if isIndent {
			word += string(r)
			continue
		}

		if r > unicode.MaxLatin1 {
			switch width.LookupRune(r).Kind() {
			case width.EastAsianAmbiguous, width.EastAsianWide, width.EastAsianFullwidth:
				n = 2
			}
		}

		w += n
		keep := strings.IndexRune(KeepPunct, next) != -1
		// 多字节及时换行
		if n != 1 {
			if keep || w < limit {
				last += word + string(r)
			} else if w == limit {
				wrap += prefix + last + word + string(r) + nl
				w, last = 0, ""
			} else {
				wrap += strings.TrimRight(prefix+last+word, " ") + nl
				w, last = 0, string(r)
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
			wrap += strings.TrimRight(prefix+last+word+string(r), " ") + nl
			w, last, word = 0, "", ""
		} else {
			wrap += strings.TrimRight(prefix+last, " ") + nl
			w, last, word = n, "", word+string(r)
		}
	}
	if word != "" || last != "" {
		wrap += prefix + last + word
	}
	if next != 0 {
		wrap += string(next)
	}
	return
}

func fprint(output io.Writer, i ...interface{}) (err error) {
	_, err = fmt.Fprint(output, i...)
	return err
}

var config = printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}

// Godoc 仿 godoc 风格向 output 输出已排序的 ast.File.
func Godoc(output io.Writer, paths string, fset *token.FileSet, file *ast.File) (err error) {
	text := file.Name.String()
	if err = fprint(output, "PACKAGE DOCUMENTATION\n\npackage ", text, nl); err != nil {
		return
	}

	if pos := strings.IndexByte(paths, ':'); pos == -1 {
		text = `    import "` + paths + `"`
	} else if paths[pos:] == ":main" {
		// BUG: 可能是 +build ignore
		text = `    EXECUTABLE PROGRAM IN PACKAGE ` + paths[:pos]
	} else if paths[pos:] == ":test" || strings.HasSuffix(paths, "_test") {
		text = `    go test ` + paths[:pos]
	}

	if err = fprint(output, text, nl); err == nil && file.Doc != nil {
		if err = fprint(output, nl); err == nil {
			err = ToText(output, file.Doc.Text())
		}
	}

	if err == nil && len(file.Imports) != 0 {
		err = fprint(output, "\nIMPORTS\n\n", ImportsString(file.Imports))
	}
	if err != nil {
		return
	}

	step := ImportNum
	for _, decl := range file.Decls {
		num := NodeNumber(decl)
		if num == ImportNum {
			continue
		}

		if num == FuncNum || num == MethodNum {
			fdecl := decl.(*ast.FuncDecl)
			if step != num {
				if num == FuncNum {
					err = fprint(output, "\nFUNCTIONS\n")
				} else {
					err = fprint(output, "\nMETHODS\n")
				}
				if err != nil {
					return
				}
				step = num
			}
			err = fprint(output, nl, FuncLit(fdecl), nl)
			if err == nil && fdecl.Doc != nil {
				err = ToText(output, fdecl.Doc.Text())
			}
			if err != nil {
				return
			}
			continue
		}
		genDecl := decl.(*ast.GenDecl)
		if len(genDecl.Specs) == 0 {
			continue
		}

		if step != num {
			step = num
			switch num {
			case TypeNum:
				err = fprint(output, "\nTYPES\n")
			case ConstNum:
				err = fprint(output, "\nCONSTANTS\n")
			case VarNum:
				err = fprint(output, "\nVARIABLES\n")
			}
			if err != nil {
				return
			}
		}
		docGroup := genDecl.Doc
		genDecl.Doc = nil
		if err = fprint(output, nl); err != nil {
			return
		}
		if err = config.Fprint(output, fset, genDecl); err != nil {
			return
		}
		if err = fprint(output, nl); err != nil {
			return
		}
		if err = ToText(output, docGroup.Text()); err != nil {
			return
		}
		genDecl.Doc = docGroup
	}
	return
}

// DocGo 以 go source 风格向 output 输出已排序的 ast.File.
func DocGo(output io.Writer, paths string, fset *token.FileSet, file *ast.File) (err error) {
	err = fprint(output, "// +build ingore\n\n")

	if err == nil && file.Doc != nil {
		err = ToSource(output, file.Doc.Text())
	}
	if err != nil {
		return
	}

	text := file.Name.String()

	if pos := strings.IndexByte(paths, ':'); pos != -1 {
		paths = paths[:pos]
	}

	if text == "main" {
		text += " // go get " + paths
	} else if text == "test" || strings.HasSuffix(text, "_test") {
		text += " // go test " + paths
	} else {
		text += ` // import "` + paths + `"`
	}

	err = fprint(output, "package ", text, nl+nl)

	if err == nil && len(file.Imports) != 0 {
		err = fprint(output, ImportsString(file.Imports), nl)
	}
	if err != nil {
		return
	}

	for _, decl := range file.Decls {
		num := NodeNumber(decl)
		if num == ImportNum {
			continue
		}

		if num == FuncNum || num == MethodNum {
			fdecl := decl.(*ast.FuncDecl)
			if fdecl.Doc != nil {
				if err = ToSource(output, fdecl.Doc.Text()); err != nil {
					return
				}
			}

			err = fprint(output, FuncLit(fdecl), nl+nl)

			if err != nil {
				return
			}
			continue
		}
		genDecl := decl.(*ast.GenDecl)
		if len(genDecl.Specs) == 0 {
			continue
		}

		docGroup := genDecl.Doc
		genDecl.Doc = nil
		if err = ToSource(output, docGroup.Text()); err != nil {
			return
		}
		if err = config.Fprint(output, fset, genDecl); err != nil {
			return
		}
		if err = fprint(output, nl+nl); err != nil {
			return
		}
		genDecl.Doc = docGroup
	}
	return
}
