package docu

import (
	"bytes"
	"go/ast"
	"go/doc"
	"go/token"
	"path"
	"strings"
	"text/template"
)

// Data 为模板提供执行数据.
type Data struct {
	Docu       *Docu
	ImportPath string // 提取到的传统 ImportPath
	Key        string // 模板将要要处理的
	Ext        string // 输出文件扩展名
	// 方便起见包含了声明类型常量
	IMPORT, CONST, VAR, TYPE, FUNC, METHOD, OTHER int

	buf    bytes.Buffer // 仅供模板内部处理文本用
	filter func(*ast.File) bool
}

// NewData 返回需要自建立 Data.Docu 的 Data 对象.
func NewData() *Data {
	return &Data{
		IMPORT: ImportNum,
		CONST:  ConstNum,
		VAR:    VarNum,
		TYPE:   TypeNum,
		FUNC:   FuncNum,
		METHOD: MethodNum,
	}
}

func (d *Data) Parse(path string, source interface{}) (paths []string, err error) {
	d.Ext = ""
	d.ImportPath = ""

	paths, err = d.Docu.Parse(path, source)
	if err != nil || len(paths) == 0 {
		return
	}
	d.ImportPath = paths[0]
	if pos := strings.Index(d.ImportPath, "::"); pos != -1 {
		d.ImportPath = d.ImportPath[:pos]
	}
	return
}

func (d *Data) SetFilter(filter func(*ast.File) bool) {
	d.filter = filter
}

// File 返回 MergePackageFiles d.Key 的值
func (d *Data) File() *ast.File {
	f := d.Docu.MergePackageFiles(d.Key)
	if d.filter != nil {
		d.filter(f)
	}
	return f
}

// Type 设置 d.Ext
func (d *Data) Type(ext string) string {
	d.Ext = ext
	return ""
}

// Code 返回 decl 的代码, 支持 Const,Var,Type,Func
func (d *Data) Code(decl ast.Decl) string {
	num := NodeNumber(decl)
	if num == FuncNum || num == MethodNum {
		return FuncLit(decl.(*ast.FuncDecl))
	}
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok {
		return ""
	}
	if len(genDecl.Specs) == 0 {
		return ""
	}

	docGroup := genDecl.Doc
	genDecl.Doc = nil
	d.buf.Truncate(0)
	config.Fprint(&d.buf, d.Docu.FileSet, genDecl)
	genDecl.Doc = docGroup
	return d.buf.String()
}

// Text 返回 decl 的注释, 支持 Const,Var,Type,Func
func (d *Data) Text(decl ast.Decl) string {
	num := NodeNumber(decl)
	if num == FuncNum || num == MethodNum {
		fdecl := decl.(*ast.FuncDecl)
		return fdecl.Doc.Text()
	}
	if num != ConstNum && num != VarNum && num != TypeNum {
		return ""
	}
	genDecl := decl.(*ast.GenDecl)
	return genDecl.Doc.Text()
}

// Fold 利用 doc.ToText 对文档 text 进行折叠.
func (d *Data) Fold(text string) string {
	d.buf.Truncate(0)
	doc.ToText(&d.buf, text, "", "    ", 1<<32)
	return d.buf.String()
}

// FuncsMap 是默认的 template.FuncMap
var FuncsMap = template.FuncMap{
	"base":                 path.Base,
	"progress":             TranslationProgress,
	"canonicalImportPaths": CanonicalImportPaths,
	"license":              License,
	"nodeNum":              NodeNumber,
	"lineWrap":             LineWrapper,
	"identLit":             DeclIdentLit,
	"prevComment": func(comments []*ast.CommentGroup, pos token.Pos) *ast.CommentGroup {
		// prevComment 在 comments 中查找 pos 的上一个 CommentGroup.
		// 返回 nil 表示未找到.
		i := findCommentPrev(pos-2, comments)
		if i == -1 {
			return nil
		}
		return comments[i]
	},
	"imports": func(file *ast.File) string {
		// 返回 file 的 import 代码
		return ImportsString(file.Imports)
	},
	"wrap": func(text string) string {
		// 纯文本无前导缩进
		return WrapComments(text, "", 1<<32)
	},
	"starLess": func(lit string) string {
		// 去掉 lit 前面的星号
		if lit == "" || lit[0] != '*' {
			return lit
		}
		return lit[1:]
	},
	"sourceWrap": func(text string) string {
		// go source 风格, 前导 "// "
		return WrapComments(text, "// ", 1<<32)
	},
	"decls": func(decls []ast.Decl, num int) []ast.Decl {
		// 返回已经排序的 decls 中指定节点类型 num 的声明
		if num >= OtherNum {
			return nil
		}
		first := -1
		last := len(decls)
		// 返回
		for i, decl := range decls {
			if decl == nil {
				continue
			}
			if num == NodeNumber(decl) {
				if first == -1 {
					first = i
				}
			} else if first != -1 {
				last = i
				break
			}
		}
		if first == -1 {
			return nil
		}
		return decls[first:last]
	},
	"indexConstructor": func(decls []ast.Decl, typeLit string) int {
		// 未来实现了常规排序后, 不推荐使用此方法
		for i, n := range decls {
			num := NodeNumber(n)
			if num == FuncNum {
				if isConstructor(n.(*ast.FuncDecl), typeLit) {
					return i
				}
			}
			if num > FuncNum {
				break
			}
		}
		return -1
	},
	"methods": func(decls []ast.Decl, typeLit string) []ast.Decl {
		// 未来实现了常规排序后, 不推荐使用此方法
		first := -1
		last := len(decls)
		for i, n := range decls {
			if n == nil {
				continue
			}
			num := NodeNumber(n)
			if num == MethodNum {
				lit := RecvIdentLit(n.(*ast.FuncDecl))
				is := lit == typeLit ||
					lit != "" && lit[0] == '*' && lit[1:] == typeLit

				if first == -1 {
					if is {
						first = i
					}
				} else if !is {
					last = i
					break
				}

			} else if first != -1 || num > MethodNum {
				last = i
				break
			}
		}
		if first == -1 {
			return nil
		}
		return decls[first:last]
	},
	"clear": func(decls []ast.Decl, pos int) string {
		// 未来实现了常规排序后, 不推荐使用此方法
		if pos >= 0 && pos < len(decls) {
			decls[pos] = nil
		}
		return ""
	},
	"trimRight": func(decls []ast.Decl) []ast.Decl {
		// 未来实现了常规排序后, 不推荐使用此方法
		for i, n := range decls {
			if n != nil {
				return decls[i:]
			}
		}
		return nil
	},
}

const DefaultTemplate = MarkdownTemplate
