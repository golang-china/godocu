package docu

import (
	"bytes"
	"errors"
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
// 该方法会移除 GoDocu 分割线 ___GoDocu_Dividing_line___
func FormatComments(output io.Writer, text, prefix string, limit int) (err error) {
	n := strings.Index(text, "___GoDocu_Dividing_line___")
	if n > 1 {
		if text[n-1] != '\n' && text[n+26] != '\n' {
			return errors.New("invalid ___GoDocu_Dividing_line___")
		}
		// 需要剔除右侧空白, 可用 merge builtin 测试
		err = FormatComments(output, strings.TrimRightFunc(text[:n-1], unicode.IsSpace), prefix, limit)
		if err == nil {
			if _, err = io.WriteString(output, "\n"); err == nil {
				err = FormatComments(output, text[n+27:], prefix, limit)
			}
		}
		return
	}
	if text != "" {
		var buf bytes.Buffer
		// 利用 ToText 的 preIndent 功能合并连续行
		if !IsWrapped(text, limit) {
			doc.ToText(&buf, text, "", "    ", 1<<32)
			text = buf.String()
		} else {
			limit = 1 << 32
		}
		_, err = io.WriteString(output, LineWrapper(text, prefix, limit))

	}
	return
}

// UrlPos 识别 text 第一个网址出现的位置.
func UrlPos(text string) (pos int) {
	pos = strings.Index(text, "://")
	if pos <= 0 {
		return -1
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
// tab 按四个长度计算, 多字节按两个长度计算.
func IsWrapped(text string, limit int) bool {
	w := 0
	for _, r := range text {
		switch r {
		case '\n':
			w = 0
		case '\t':
			w += 4
		default:
			if r > unicode.MaxLatin1 {
				w += 2
			} else {
				w++
			}
		}
		if w > limit {
			return false
		}
	}
	return true
}

func visualWidth(text string) (w int) {
	for _, r := range text {
		if r > unicode.MaxLatin1 {
			w += 2
		} else {
			w++
		}
	}
	return
}

// KeepPunct 当这些标点符号位于行尾时, 前一个词会被折到下一行.
var KeepPunct = `,.:;?，．：；？。`

// WrapPunct 当这些标点符号位于行尾时, 会被折到下一行.
var WrapPunct = "`!*@" + `"'[(“（［`

func runeWidth(r rune) int {
	if r == 0 {
		return 0
	}
	if r > unicode.MaxLatin1 {
		switch width.LookupRune(r).Kind() {
		case width.EastAsianAmbiguous, width.EastAsianWide, width.EastAsianFullwidth:
			return 2
		}
	}
	return 1
}

// LineWrapper 把 text 非缩进行超过显示长度 limit 的行插入换行符 "\n".
// 细节:
//	text   行间 tab 按 4 字节宽度计算.
//	prefix 为每行前缀字符串.
//	limit  的长度不包括 prefix 的长度.
//	返回 wrap 的尾部带换行
func LineWrapper(text string, prefix string, limit int) (wrap string) {
	// go scanner 已经剔除了 '\r', 统一为 '\n' 风格
	const nl = "\n"
	var r, next rune
	var isIndent bool
	var last, word string // 最后一行前部和尾部单词
	if text == "" {
		return ""
	}
	n, w, nw, ww := 0, 0, 0, 0
	for _, r = range text {
		// 预读取一个
		r, next = next, r
		if r == 0 {
			// 剔除首行缩进
			if wrap == "" && (next == ' ' || next == '\t' || w == 0 && next == '\n') {
				next = 0
				continue
			}
			nw = runeWidth(next)
			continue
		}
		switch r {
		case '\n':
			if wrap != "" || last != "" || word != "" {
				wrap += strings.TrimRight(prefix+last+word, " ") + nl
			}
			w, ww, last, word = 0, 0, "", ""
			isIndent = false
			nw = runeWidth(next)
			continue
		case '\t':
			// tab 缩进替换为 4 空格, 保持行间 tab
			if last == "" || len(word) >= 4 && word[:4] == "    " {
				w, word = w+4, word+"    "
				isIndent = true
			} else {
				w, last, word = w/4*4+4, last+"\t", ""
			}
			continue
		case ' ':
			// 行首连续两个空格算做缩进
			if next == ' ' && last == "" {
				isIndent = true
			}
		}

		if isIndent {
			word += string(r)
			continue
		}
		n, nw = nw, runeWidth(next)

		w += n
		ww += n // word width
		word += string(r)
		// 确定下一个 rune 归属行
		if n == 2 || r == ' ' || n != nw {
			// 分离单词或者单字节多字节混合, 例如: 值value
			last += word
			word, ww = "", 0
		}

		if w == limit {
			if next == ' ' || next == '\n' {
				next = '\n'
				continue
			}
			wrap += strings.TrimRight(prefix+last, " ") + nl
			last = ""
			w = ww
			continue
		}

		if w+nw <= limit {
			continue
		}
		if next != ' ' && strings.IndexRune(KeepPunct, next) != -1 {
			wrap += strings.TrimRight(prefix+last, " ") + nl
			last = ""
			w = ww
			continue
		}

		if next != ' ' && strings.IndexRune(WrapPunct, next) != -1 {
			wrap += strings.TrimRight(prefix+last+word, " ") + nl
			last, word = "", string(next)
			w, ww = nw, nw
			next, nw = 0, 0
			continue
		}

		if pos := UrlPos(last); pos != -1 {
			if pos == 0 {
				wrap += strings.TrimRight(prefix+last, " ") + nl
				last = ""
				w = ww
			} else {
				wrap += strings.TrimRight(prefix+last[:pos], " ") + nl
				last, word = last[pos:]+word, ""
				w, ww = visualWidth(last), 0
			}
			continue
		}

		wrap += strings.TrimRight(prefix+last, " ") + nl
		last = ""
		w = 0
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

	if wrap[len(wrap)-1] != '\n' {
		wrap += "\n"
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
	} else if text = ImportPaths(file); text != "" {
		text = `    ` + text
	}
	if text != "" {
		err = fprint(output, text, nl)
	}
	if err == nil && file.Doc != nil {
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
	} else if imp := ImportPaths(file); imp != "" {
		text += ` // ` + imp
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
