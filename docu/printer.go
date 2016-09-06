package docu

import (
	"bytes"
	"go/ast"
	"go/doc"
	"go/printer"
	"go/token"
	"go/types"
	"io"
	"strings"
	"text/tabwriter"
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

var prefix = []string{"// ", "\t// ", "\t\t// "}
var indents = []string{"", "\xff\t\xff", "\xff\t\t\xff"}
var rawindents = []string{"", "\t", "\t\t"}

// Format 调用 LineWrapper 换行格式化注释 doc 输出到 output.
// indent 是 "\t" 缩进个数, 值范围为 0,1,2.
// 如果 doc 是合并文档, 包含 GoDocu_Dividing_line, 表示输出双语文档.
// 如果 doc 非双语文档且 comments 非 nil, 则在 comments 中查找并输出 OriginDoc.
func Format(output io.Writer, indent int,
	doc *ast.CommentGroup, comments []*ast.CommentGroup) (err error) {
	if doc == nil {
		return
	}
	source, text := SplitComments(doc.Text())
	if source == "" && comments != nil {
		source = OriginDoc(comments, doc).Text()
	}
	if source == "" && text == "" {
		return
	}
	if source != "" {
		source = WrapComments(source, prefix[indent], 77-indent*4)
	}
	if text != "" {
		text = WrapComments(text, prefix[indent], 77-indent*4)
	}
	tw, istw := output.(*tabwriter.Writer)
	// 防止 Wrap 后结果一样
	if source != "" && source != text {
		if istw {
			err = fprint(output, tabEscapes, source, tabEscapes, nl)
		} else {
			err = fprint(output, source, nl)
		}
	}
	if text != "" && err == nil {
		if istw {
			err = fprint(output, tabEscapes, text, tabEscapes)
		} else {
			err = fprint(output, text)
		}
	}
	if err == nil && tw != nil {
		err = tw.Flush()
	}
	return
}
func isNotSpace(r rune) bool {
	return !unicode.IsSpace(r)
}

// WrapComments 对 text 连续行进行折叠后调用 LineWrapper.
func WrapComments(text, prefix string, limit int) string {
	var buf bytes.Buffer
	if text == "" {
		return ""
	}
	offset := wrappedBefor(text, limit)
	if offset == len(text) {
		limit = 1 << 32
	} else {
		doc.ToText(&buf, text, "", "\t", 1<<32)
		text = buf.String()
	}
	return LineWrapper(text, prefix, limit)
}

// wrappedBefor 返回 text 首个行长度大于 limit 的行首偏移量.
// 如果满足 limit, 返回 len(text)
// tab 按四个长度计算, 多字节按两个长度计算.
func wrappedBefor(text string, limit int) int {
	offset, w := 0, 0

	for i, r := range text {
		if w == 0 {
			offset = i
		}
		switch r {
		case '\n':
			w = 0
			w += 4
		case '\t':
			w = w/4*4 + 4
		default:
			if r > unicode.MaxLatin1 {
				w += 2
			} else {
				w++
			}
		}
		if w > limit {
			return offset
		}
	}
	return len(text)
}

// SplitComments 以 GoDocu_Dividing_line 分割 text 为两部分.
// 如果没有分割线返回 "",text
func SplitComments(text string) (string, string) {
	n := strings.Index(text, GoDocu_Dividing_line)
	if n == -1 {
		return "", text
	}
	return text[:n], text[n+len(GoDocu_Dividing_line)+1:]
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

func urlEnd(r rune) bool {
	return r == 0 || runeWidth(r) == 2 || unicode.IsSpace(r)
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
// 已过时, 未来会删除
var KeepPunct = `,.:;?，．：；？。`

// WrapPunct 当这些标点符号位于行尾时, 会被折到下一行.
// 已过时,未来会删除
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
		return w, '\n'
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

		last = text[:pos]

		wrap += strings.TrimRightFunc(prefix+last, unicode.IsSpace) + nl

		if len(text) != pos && r == '\n' {
			text = text[pos+1:]
		} else {
			text = strings.TrimLeftFunc(text[pos:], unicode.IsSpace)
		}
	}
	// 剔除前部和尾部多余的空白行
	prefix = strings.TrimRightFunc(prefix, unicode.IsSpace) + nl
	for strings.HasPrefix(wrap, prefix) {
		wrap = wrap[len(prefix):]
	}
	for strings.HasSuffix(wrap, prefix) {
		wrap = wrap[:len(wrap)-len(prefix)]
	}
	if wrap != "" && wrap[len(wrap)-1] != '\n' {
		wrap += nl
	}
	return
}

func fprint(output io.Writer, ss ...string) (err error) {
	if output != nil {
		for i := 0; err == nil && i < len(ss); i++ {
			if ss[i] != "" {
				_, err = io.WriteString(output, ss[i])
			}
		}
	}
	return
}

var config = printer.Config{Mode: printer.TabIndent, Tabwidth: 4}
var emptyfset = token.NewFileSet()
var tabEscape = []byte{tabwriter.Escape}
var tabEscapes = string(tabEscape)

func fprintExpr(w io.Writer, expr ast.Expr, before ...string) (err error) {
	if err = fprint(w, before...); err == nil {
		err = config.Fprint(w, emptyfset, expr)
	}
	return
}

// Fprint 以 go source 风格向 output 输出已排序的 ast.File.
func Fprint(output io.Writer, file *ast.File) (err error) {
	var text string
	var comments []*ast.CommentGroup

	if IsGodocuFile(file) {
		comments = file.Comments
	}

	if text, _ = License(file); text != "" {
		err = fprint(output, LineWrapper(text, "// ", 77), nl)
	}
	if err == nil {
		err = fprint(output, "// +build ingore\n\n")
	}

	if err == nil {
		err = Format(output, 0, file.Doc, comments)
	}
	if err != nil {
		return
	}

	text = file.Name.String()

	if imp := CanonicalImportPaths(file); imp != "" {
		text += ` // ` + imp
	}

	err = fprint(output, "package ", text, nl+nl)

	if err == nil && len(file.Imports) != 0 {
		err = fprint(output, ImportsString(file.Imports), nl)
	}

	if err != nil {
		return
	}
	for _, node := range file.Decls {
		switch n := node.(type) {
		case *ast.GenDecl:
			switch n.Tok {
			default:
				continue
			case token.CONST, token.VAR:
				err = FprintGenDecl(output, n, comments)
			case token.TYPE:
				err = FprintGenDecl(output, n, comments)
			}
		case *ast.FuncDecl:
			err = FprintFuncDecl(output, n, comments)
		}
		if err == nil {
			err = fprint(output, nl)
		}
		if err != nil {
			break
		}
	}
	return
}

// FprintFuncDecl 向 w 输出顶级函数声明 fn. comments 用于输出双语文档.
func FprintFuncDecl(w io.Writer, fn *ast.FuncDecl, comments []*ast.CommentGroup) (err error) {
	err = Format(w, 0, fn.Doc, comments)
	if err == nil {
		err = fprint(w, MethodLit(fn), nl)
	}
	return
}

// FprintGenDecl 向 w 输出顶级声明 decl. comments 用于输出双语文档.
func FprintGenDecl(w io.Writer, decl *ast.GenDecl, comments []*ast.CommentGroup) (err error) {
	if decl == nil || len(decl.Specs) == 0 || decl.Tok == token.IMPORT {
		return
	}
	if err = Format(w, 0, decl.Doc, comments); err != nil {
		return
	}

	switch decl.Tok {
	case token.CONST:
		err = fprint(w, "const ")
	case token.VAR:
		err = fprint(w, "var ")
	case token.TYPE:
		err = fprint(w, "type ")
	}

	if err != nil {
		return
	}

	indent := 0
	tw := NewWriter(w)
	if decl.Lparen.IsValid() {
		indent = 1
		err = fprint(tw, "(\n")
	}

	out := false
	for _, spec := range decl.Specs {
		if spec == nil {
			continue
		}
		switch decl.Tok {
		case token.CONST, token.VAR:
			vs := spec.(*ast.ValueSpec)
			if out && vs.Doc != nil {
				if err = fprint(tw, nl); err != nil {
					break
				}
			}
			err = FprintValueSpec(tw, indent, vs, comments)
		case token.TYPE:
			vs := spec.(*ast.TypeSpec)
			if out {
				if err = fprint(tw, "\f"); err != nil {
					break
				}
			}
			err = FprintTypeSpec(tw, indent, vs, comments)
		}

		if err != nil {
			break
		}
		out = true
	}

	if out && err == nil && decl.Lparen.IsValid() {
		err = fprint(tw, ")\n")
	}

	if err == nil {
		err = tw.Flush()
	}
	return
}

// NewWriter 返回适用 Docu 的 tabwriter.Writer 实例.
func NewWriter(w io.Writer) *tabwriter.Writer {
	tw, ok := w.(*tabwriter.Writer)
	if !ok {
		tw = tabwriter.NewWriter(w, 0, 4, 1, ' ',
			tabwriter.StripEscape|tabwriter.DiscardEmptyColumns)
	}
	return tw
}

// FprintValueSpec 向 w 输出 vs. indent 是 tab 缩进个数, comments 用于输出双语文档.
func FprintValueSpec(w *tabwriter.Writer, indent int,
	vs *ast.ValueSpec, comments []*ast.CommentGroup) (err error) {

	if err = Format(w, indent, vs.Doc, comments); err == nil {
		err = fprint(w, indents[indent], IdentsLit(vs.Names))
	}
	if err != nil {
		return
	}

	if vs.Type != nil {
		err = fprintExpr(w, vs.Type, "\v")
	} else if len(vs.Values) != 0 || vs.Comment != nil {
		err = fprint(w, "\v")
	} else {
		err = fprint(w, nl)
		return
	}

	for i, expr := range vs.Values {
		if i == 0 {
			err = fprintExpr(w, expr, "\v= ")
		} else {
			err = fprintExpr(w, expr, ", ")
		}
		if err != nil {
			return
		}
	}
	if err != nil {
		return
	}
	if vs.Comment != nil {
		if len(vs.Values) == 0 {
			if err = fprint(w, "\v"); err != nil {
				return
			}
		}
		err = fprint(w, "\v", tabEscapes,
			trimNL(comments, vs.Comment), tabEscapes, nl)
	} else {
		err = fprint(w, nl)
	}
	return
}

// FprintTypeSpec 向 w 输出 ts. indent 是 tab 缩进个数, comments 用于输出双语文档.
func FprintTypeSpec(w *tabwriter.Writer, indent int,
	ts *ast.TypeSpec, comments []*ast.CommentGroup) (err error) {

	if err = Format(w, indent, ts.Doc, comments); err == nil {
		err = fprint(w, indents[indent], ts.Name.String())
	}
	if err != nil {
		return
	}

	if st, ok := ts.Type.(*ast.StructType); ok {
		fprint(w, " struct {\f")
		if err = FprintFieldList(w, indent+1, st.Fields, comments); err == nil {
			err = fprint(w, indents[indent], "}\f")
		}
		return
	}

	if st, ok := ts.Type.(*ast.InterfaceType); ok {
		fprint(w, " interface {\f")
		if err = FprintMethods(w, indent+1, st.Methods, comments); err == nil {
			err = fprint(w, indents[indent], "}\f")
		}
		return
	}

	err = fprintExpr(w, ts.Type, "\v")
	if err != nil {
		return
	}
	if ts.Comment != nil {
		err = fprint(w, "\v", tabEscapes,
			trimNL(comments, ts.Comment), tabEscapes, nl)
	} else {
		err = fprint(w, nl)
	}
	return
}

// FprintFieldList 向 w 输出 fields. indent 是 tab 缩进个数, comments 用于输出双语文档.
func FprintFieldList(w *tabwriter.Writer, indent int, fields *ast.FieldList, comments []*ast.CommentGroup) (err error) {
	for i, field := range fields.List {
		if field.Doc != nil {
			// 注释前加换行
			if i != 0 {
				fprint(w, nl)
			}
			if err = Format(w, indent, field.Doc, comments); err != nil {
				break
			}
		}
		if len(field.Names) == 0 {
			err = fprintExpr(w, field.Type, indents[indent])
		} else {
			err = fprintExpr(w, field.Type, indents[indent], IdentsLit(field.Names), "\v")
		}
		if err == nil && field.Tag != nil {
			err = fprint(w, " ", field.Tag.Value)
		}

		if err == nil && field.Comment != nil {
			err = fprint(w, "\v", tabEscapes,
				trimNL(comments, field.Comment), tabEscapes, nl)
		} else if err == nil {
			err = fprint(w, nl)
		}
		if err != nil {
			break
		}
	}
	if err == nil {
		err = w.Flush()
	}
	return
}

// FprintMethods 向 w 输出接口的 methods. indent 是 tab 缩进个数, comments 用于输出双语文档.
func FprintMethods(w *tabwriter.Writer, indent int, methods *ast.FieldList, comments []*ast.CommentGroup) (err error) {
	for i, field := range methods.List {
		if field.Doc != nil {
			// 注释前加换行
			if i != 0 {
				fprint(w, nl)
			}
			if err = Format(w, indent, field.Doc, comments); err != nil {
				return
			}
		}
		if ftyp, isFtyp := field.Type.(*ast.FuncType); isFtyp {
			// method
			lit := FieldListLit(ftyp.Results)

			if lit != "" && (len(ftyp.Results.List) > 1 ||
				len(ftyp.Results.List[0].Names) != 0) {
				lit = " (" + lit + ")"
			}
			fprint(w, indents[indent], field.Names[0].String(),
				"("+FieldListLit(ftyp.Params)+")", lit)
		} else {
			// embedded interface
			fprint(w, indents[indent], types.ExprString(field.Type))
		}

		if field.Comment != nil {
			fprint(w, "\t", tabEscapes,
				trimNL(comments, field.Comment), tabEscapes, nl)
		} else {
			fprint(w, nl)
		}
	}
	err = w.Flush()
	return
}

func trimNL(comments []*ast.CommentGroup, comment *ast.CommentGroup) string {
	ClearComment(comments, comment)
	isblock := len(comment.List) != 0 && comment.List[0].Text[1] == '*'

	s := comment.Text()
	l := len(s)
	if l > 0 && s[l-1] == '\n' {
		s = s[:l-1]
	}
	if len(s) != 0 {
		if isblock {
			s = "/*" + s + "*/"
		} else {
			s = "// " + s
		}
	}
	return s
}
