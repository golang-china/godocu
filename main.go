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

func flagParse() ([]string, docu.Mode) {
	var gopath string
	var mode docu.Mode

	flag.Usage = usage
	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "Go root directory")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "specifies gopath")

	if *flag.Bool("u", false, "show unexported symbols as well as exported") {
		mode |= docu.ShowUnexported
	}

	if *flag.Bool("cmd", false, "show symbols with package docs even if package is a command") {
		mode |= docu.ShowCMD
	}

	if *flag.Bool("test", false, "show symbols with package docs even if package is a testing") {
		mode |= docu.ShowTest
	}

	flag.Parse()
	pkgs := flag.Args()
	if len(pkgs) == 0 {
		flag.Usage()
	}

	if gopath != "" {
		docu.GOPATHS = filepath.SplitList(gopath)
	}
	return pkgs, mode
}

func main() {
	var err error
	paths, mode := flagParse()
	du := docu.New(mode)
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
