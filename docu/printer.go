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
	var source string
	source, text = SplitComments(text)
	if source != "" {
		_, err = io.WriteString(output, WrapComments(source, prefix, limit)+"\n")
	}
	if text != "" && err == nil {
		_, err = io.WriteString(output, WrapComments(text, prefix, limit))
	}
	return
}
func isNotSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

// WrapComments 优化调用 LineWrapper.
// 如果 !IsWrapped(text, limit) 对 text 连续行进行折叠.
func WrapComments(text, prefix string, limit int) string {
	var buf bytes.Buffer
	if text == "" {
		return ""
	}
	if !IsWrapped(text, limit) {
		doc.ToText(&buf, text, "", "    ", 1<<32)
		text = buf.String()
	} else {
		limit = 1 << 32
	}
	return LineWrapper(text, prefix, limit)
}

// SplitComments 以 "___GoDocu_Dividing_line___" 分割 text 为两部分.
// 如果没有分割线返回 "",text
func SplitComments(text string) (string, string) {
	n := strings.Index(text, "___GoDocu_Dividing_line___")
	if n == -1 {
		return "", text
	}
	return strings.TrimRightFunc(text[:n], unicode.IsSpace),
		strings.TrimLeftFunc(text[n+26:], unicode.IsSpace)
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

func firstWidth(text string) (w int) {
	for _, r := range text {
		if r == '\n' {
			return
		}
		if r > unicode.MaxLatin1 {
			w += 2
		} else {
			w++
		}
	}
	return
}

// KeepPunct 当这些标点符号位于折行处时, 前一个词会被折到下一行.
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

// 返回 text 第一行内不超过 limit 的位置
func limitPos(text string, limit int) (int, rune) {
	const keepPunct = "`!*@" + `,.:;?，．：；？。"'[(“（［`

	var s, ps, ww int
	var pr, sr rune
	w := strings.IndexByte(text, '\n')
	if text[0] == '\t' || strings.HasPrefix(text, "    ") {
		if w == -1 {
			return len(text), 0
		}
		return w, 0
	}

	w, ww = 0, 1
	for i, r := range text {
		if r == '\n' {
			return i, r
		}
		ps, pr = s, sr
		if r == '\t' {
			w = w/4*4 + 4
			s, sr = i, r
		} else {
			if r == ' ' || r == '　' {
				s, sr = i, r
			}
		}
		width := runeWidth(r)
		if width == 2 || ww != width {
			s, sr = i, r
		}
		ww = width
		// 连续长字符串
		if s == 0 || w+ww <= limit {
			w += ww
			continue
		}
		if s != i || ps == 0 || strings.IndexRune(keepPunct, r) == -1 {
			return s, sr
		}
		// 回退单词, 保持标点符号
		return ps, pr
	}
	return len(text), 0
}

func urlEnd(r rune) bool {
	return r == 0 || runeWidth(r) == 2 || unicode.IsSpace(r)
}

// LineWrapper 把 text 非缩进行超过显示长度 limit 的行插入换行符 "\n".
// 细节:
//	text   行间 tab 按 4 字节宽度计算.
//	prefix 为每行前缀字符串.
//	limit  的长度不包括 prefix 的长度.
//	返回 wrap 的尾部带换行
func LineWrapper(text string, prefix string, limit int) (wrap string) {
	for len(text) != 0 {
		last := ""
		pos, r := limitPos(text, limit)

		if pos <= limit && pos != len(text) && !urlEnd(r) {
			// 保证网址完整性
			i := UrlPos(text[:pos])
			if i != -1 {
				pos = strings.IndexFunc(text[i:], unicode.IsSpace)
				if pos == -1 {
					pos = len(text) - i
				}
				if i != 0 {
					last = text[:i]
					wrap += strings.TrimRightFunc(prefix+last, unicode.IsSpace) + nl
				}
				last, text = text[i:pos], text[i+pos:]
				wrap += strings.TrimRightFunc(prefix+last, unicode.IsSpace) + nl

				if len(text) != pos && text[pos] == '\n' {
					text = text[pos+1:]
				} else {
					text = strings.TrimLeftFunc(text[pos:], unicode.IsSpace)
				}
				continue
			}
		}

		if text[0] == '\t' {
			i := 0
			for ; i < len(text); i++ {
				if text[i] != '\t' {
					break
				}
				last += "    "
			}
			last += text[i:pos]
		} else {
			last = text[:pos]
		}

		wrap += strings.TrimRightFunc(prefix+last, unicode.IsSpace) + nl

		if len(text) != pos && text[pos] == '\n' {
			text = text[pos+1:]
		} else {
			text = strings.TrimLeftFunc(text[pos:], unicode.IsSpace)
		}

	}

	prefix = strings.TrimRightFunc(prefix, unicode.IsSpace)
	if prefix != "" {
		prefix += nl
		for strings.HasPrefix(wrap, prefix) {
			wrap = wrap[len(prefix):]
		}
	}
	if wrap == "" || wrap[len(wrap)-1] != '\n' {
		wrap += nl
	}
	return wrap
}

// 老算法
func _lineWrapper(text string, prefix string, limit int) (wrap string) {
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
			// NOTE(achun): 不应该剔除首行缩进, 那可能是作者强调的信息
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
	} else if text = CanonicalImportPaths(file); text != "" {
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
	} else if imp := CanonicalImportPaths(file); imp != "" {
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
