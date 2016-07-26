// +build go1.5

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
	var u, cmd, test, source bool

	flag.Usage = usage
	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "Go root directory")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "specifies gopath")
	flag.BoolVar(&u, "u", false, "show unexported symbols as well as exported")
	flag.BoolVar(&cmd, "cmd", false, "show symbols with package docs even if package is a command")
	flag.BoolVar(&test, "test", false, "show symbols with package docs even if package is a testing")
	flag.BoolVar(&source, "go", false, "prints a formatted string to standard output as Go source code")

	flag.Parse()

	if u {
		mode |= docu.ShowUnexported
	}
	if cmd {
		mode |= docu.ShowCMD
	}

	if test {
		mode |= docu.ShowTest
	}
	if source {
		mode |= 1 << 30
	}

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
	godoc := docu.Godoc
	if mode&(1<<30) != 0 {
		godoc = docu.DocGo
		mode -= 1 << 30
	}
	du := docu.New(mode)
	for _, path := range paths {
		err = du.Parse(path, nil)
		if err != nil {
			log.Fatal(err)
		}
	}
	fset := du.FileSet
	for paths, pkg := range du.Astpkg {
		err = godoc(os.Stdout, paths, fset, pkg)
		if err != nil {
			log.Fatal(err)
		}
	}
}
