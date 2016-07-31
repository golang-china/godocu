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
	if text != "" {
		var buf bytes.Buffer
		// 利用 ToText 的 preIndent 功能先缩成一行
		if !IsWrapped(text, limit) {
			doc.ToText(&buf, text, "", "    ", 1<<32)
			text = buf.String()
		}
		_, err = io.WriteString(output, LineWrapper(text, prefix, limit))
	}
	return
}

// UrlPos 识别 text 第一个网址出现的位置.
func UrlPos(text string) (pos int) {
	pos = strings.Index(text, "://")
	if pos == -1 {
		return
	}
	pos--
	for pos != 0 {
		if (text[pos] >= 'a' && text[pos] <= 'z') || text[pos] == '-' {
			pos--
		} else {
			pos++
			break
		}
	}
	return
}

// IsWrapped 检查 text 每一行的长度都小于等于 limit.
func IsWrapped(text string, limit int) bool {
	n, w := 0, 0
	// 检查是否已经排好版
	for n != -1 {
		text = text[w:]
		n = strings.IndexByte(text, '\n')
		if n == -1 {
			w = len(text)
		} else {
			w = n
		}
		if w >= limit {
			return false
		}
		w = n + 1
	}
	return true
}

// KeepPunct 是 LineWrapper 进行换行时行尾要保留标点符号
var KeepPunct = `,.:;?，．：；？。`

// LineWrapper 把 text 非缩进行超过显示长度 limit 的行插入换行符 "\n".
// 细节:
//	text   行间 tab 按 4 字节宽度计算.
//	prefix 为每行前缀字符串.
//	limit  的长度不包括 prefix 的长度.
//	位于换行处的标点符被保留.
//	移除 GoDocu 分割线
func LineWrapper(text string, prefix string, limit int) (wrap string) {
	const nl = "\n"
	var lf, r, next rune
	var isIndent bool
	var last, word string // 最后一行前部和尾部单词
	n, w := 0, 0

	n = strings.Index(text, "___GoDocu_Dividing_line___")
	if n > 1 && (text[n-1] == '\n' || text[n-1] == '\r') &&
		(text[n+26] == '\n' || text[n+26] == '\r') {
		// 需要剔除右侧空白, 可用 merge builtin 测试
		return LineWrapper(strings.TrimRightFunc(text[:n-1], unicode.IsSpace), prefix, limit) + "\n\n" +
			LineWrapper(text[n+27:], prefix, limit)
	}
	for _, r = range text {
		// 预读取一个
		r, next = next, r
		if r == 0 {
			continue
		}
		switch r {
		case '\r':
			if wrap != "" || last != "" || word != "" {
				wrap += strings.TrimRight(prefix+last+word, " ") + nl
			}
			w, last, word = 0, "", ""
			isIndent, lf = false, r
			continue
		case '\n':
			if lf != '\r' {
				if wrap != "" || last != "" || word != "" {
					wrap += strings.TrimRight(prefix+last+word, " ") + nl
				}
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
			// 识别网址
			if w > limit {
				pos := UrlPos(last)
				if pos == 0 {
					wrap += strings.TrimRight(prefix+last, " ") + nl
					w = 0
				} else if pos != -1 {
					wrap += strings.TrimRight(prefix+last[:pos], " ") + nl
					last = last[pos:]
					w = len(last)
					if w > limit {
						wrap += strings.TrimRight(prefix+last, " ") + nl
						w, last = 0, ""
					}
				}
			}
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
		if wrap == "" {
			wrap = prefix + string(next)
		} else {
			wrap += string(next)
		}
	}
	return
}

func fprint(output io.Writer, i ...interface{}) (err error) {
	if output != nil {
		_, err = fmt.Fprint(output, i...)
	}
	return err
}

var config = printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}

// Godoc 仿 godoc 风格向 output 输出已排序的 ast.File.
func Godoc(output io.Writer, paths string, fset *token.FileSet, file *ast.File) (err error) {
	text := file.Name.String()
	if err = fprint(output, "PACKAGE DOCUMENTATION\n\npackage ", text, nl); err != nil {
		return
	}

	if pos := strings.LastIndex(paths, "::"); pos != -1 {
		paths = paths[:pos]
	}

	if text == "main" {
		// BUG: 可能是 +build ignore
		text = `    EXECUTABLE PROGRAM IN PACKAGE ` + paths
	} else if text == "test" || strings.HasSuffix(text, "_test") {
		text = `    go test ` + paths
	} else {
		text = `    import "` + paths + `"`
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
	if err != nil {
		return
	}

	if text = License(file); text != "" {
		if err = fprint(output, "\nLICENSE\n\n"); err == nil {
			err = ToText(output, text)
		}
	}

	return
}

// DocGo 以 go source 风格向 output 输出已排序的 ast.File.
func DocGo(output io.Writer, paths string, fset *token.FileSet, file *ast.File) (err error) {
	var text string
	if text = License(file); text != "" {
		if err = ToSource(output, text); err == nil {
			err = fprint(output, nl)
		}
	}
	if err == nil {
		err = fprint(output, "// +build ingore\n\n")
	}

	if err == nil && file.Doc != nil {
		err = ToSource(output, file.Doc.Text())
	}
	if err != nil {
		return
	}

	text = file.Name.String()

	if pos := strings.LastIndex(paths, "::"); pos != -1 {
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
