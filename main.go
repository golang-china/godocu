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
	fmt.Fprintln(os.Stderr,
		`usage: godocu package [target]
         target       the directory as an absolute base path of docs.
                      the path for output if not set -diff.`,
	)
	flag.PrintDefaults()
	os.Exit(2)
}

func flagParse() ([]string, docu.Mode) {
	var gopath string
	var mode docu.Mode
	var u, cmd, test, source, diff bool

	flag.Usage = usage
	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "Go root directory")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "specifies gopath")
	flag.BoolVar(&u, "u", false, "show unexported symbols as well as exported")
	flag.BoolVar(&cmd, "cmd", false, "show symbols with package docs even if package is a command")
	flag.BoolVar(&test, "test", false, "show symbols with package docs even if package is a testing")
	flag.BoolVar(&source, "go", false, "prints a formatted string to standard output as Go source code")
	flag.BoolVar(&diff, "diff", false, "list different of package of target-path docs")

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
	if diff {
		mode |= 1 << 31
	}

	paths := flag.Args()
	if len(paths) == 0 {
		flag.Usage()
	}

	if len(paths) == 2 {
		fi, err := os.Stat(paths[1])
		if err != nil {
			log.Fatal("target must be an absolute base path of docs, but ", err)
		}
		if !fi.IsDir() {
			log.Fatal("target must be an absolute base path of docs")
		}
	}

	if gopath != os.Getenv("GOPATH") {
		docu.GOPATHS = filepath.SplitList(gopath)
	}
	return paths, mode
}

func main() {
	var err error
	paths, mode := flagParse()
	godoc := docu.Godoc
	if 1<<30&mode != 0 {
		godoc = docu.DocGo
		mode -= 1 << 30
	}
	diff := 1<<31&mode != 0
	if diff {
		mode -= 1 << 31
	}

	du := docu.New(mode)
	err = du.Parse(paths[0], nil)
	if err != nil {
		log.Fatal(err)
	}
	if !diff {
		for paths, pkg := range du.Astpkg {
			err = godoc(os.Stdout, paths, du.FileSet, pkg)
			if err != nil {
				log.Fatal(err)
			}
		}
		return
	}

	od := docu.New(mode)
	err = od.Parse(filepath.Join(paths[1], paths[0]), nil)
	if err != nil {
		log.Fatal(err)
	}
	for key, _ := range du.Astpkg {
		_, ok := od.Astpkg[key]
		if !ok {
			fmt.Println("[TEXT] PACKAGES", key)
			return
		}
		if !docu.Same(os.Stdout, du.MergePackageFiles(key), od.MergePackageFiles(key)) {
			fmt.Println(", on package", key)
			break
		}
	}
}
