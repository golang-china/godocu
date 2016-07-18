// +build go1.5

// coming soon....
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-china/godocu/docu"
)

const mode = ast.FilterFuncDuplicates |
	ast.FilterUnassociatedComments | ast.FilterImportDuplicates

func usage() {
	fmt.Fprint(os.Stderr, "usage: godocu package ...\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func flagParse() []string {
	var gopath string
	flag.Usage = usage
	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "Go root directory")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "specifies gopath")

	flag.Parse()
	pkgs := flag.Args()
	if len(pkgs) == 0 {
		flag.Usage()
	}

	if gopath != "" {
		docu.GOPATHS = filepath.SplitList(gopath)
	}
	return pkgs
}

func fprint(output io.Writer, fset *token.FileSet, file *ast.File) error {
	return printer.Fprint(output, fset, file)
}

func main() {
	var err error
	paths := flagParse()
	du := docu.New()
	for _, path := range paths {
		err = du.Parse(path, nil)
		if err != nil {
			log.Fatal(err)
		}
	}

	fset := du.FileSet
	for _, pkg := range du.Astpkg {
		file := ast.MergePackageFiles(pkg, mode)
		err = fprint(os.Stdout, fset, file)
		if err != nil {
			log.Fatal(err)
		}
	}
}
