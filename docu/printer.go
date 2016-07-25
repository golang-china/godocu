package docu

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/printer"
	"go/token"
	"io"
	"strings"
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
	if text != "" {
		doc.ToText(output, text, "    ", "    ", 80)
	}
}

var config = printer.Config{Mode: printer.UseSpaces, Tabwidth: 4}

// Godoc 仿 godoc 风格向 output 输出经排序的 file 文档.
func Godoc(output io.Writer, paths string, fset *token.FileSet, pkg *ast.Package) (err error) {
	file := ast.MergePackageFiles(pkg,
		ast.FilterFuncDuplicates|ast.FilterUnassociatedComments|ast.FilterImportDuplicates,
	)
	Index(file)
	fmt.Fprintln(output, "PACKAGE DOCUMENTATION\n\npackage", file.Name.String())
	ToText(output, `import `+paths)
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
				fmt.Fprint(output, "\nTYPES\n\n")
			case ConstNum:
				fmt.Fprint(output, "\nCONSTANTS\n\n")
			case VarNum:
				fmt.Fprint(output, "\nVARIABLES\n\n")
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
