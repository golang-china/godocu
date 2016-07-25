// +build go1.5

// coming soon....
package main

import (
	"flag"
	"fmt"
	"go/ast"
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
	for paths, pkg := range du.Astpkg {
		err = docu.Godoc(os.Stdout, paths, fset, pkg)
		if err != nil {
			log.Fatal(err)
		}
	}
}
