// +build go1.5

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

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

func flagParse() ([]string, docu.Mode, string) {
	var gopath, lang string
	var mode docu.Mode
	var u, cmd, test, source, diff bool

	flag.Usage = usage
	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "Go root directory")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "specifies gopath")
	flag.StringVar(&lang, "lang", "origin", "the lang pattern for the output file, form like xx[_XX]")
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

	if gopath != os.Getenv("GOPATH") {
		docu.GOPATHS = filepath.SplitList(gopath)
	}
	return paths, mode, lang
}

// 多文档输出分割线
var sp = "\n\n" + strings.Repeat("/", 80) + "\n\n"

func main() {
	var source, target, ext string
	var err error
	var ok bool

	paths, mode, lang := flagParse()

	source = paths[0]
	if len(paths) > 1 {
		target = paths[1]
		fi, err := os.Stat(target)
		if err == nil && !fi.IsDir() {
			log.Fatal("target must be an absolute base path of docs")
		}
		if os.IsNotExist(err) {
			err = os.MkdirAll(target, 0777)
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	// 遍历子目录
	sub := strings.HasSuffix(source, "...")
	if sub {
		source = source[:len(source)-3]
		if source == "" {
			source = docu.GOROOT + docu.SrcElem
		}
	}

	godoc := docu.Godoc
	ext = ".text"
	if 1<<30&mode != 0 {
		godoc = docu.DocGo
		ext = ".go"
		mode -= 1 << 30
	}
	diff := 1<<31&mode != 0
	if diff {
		mode -= 1 << 31
		diffMode(mode, sub, source, target)
		return
	}

	fs := docu.Target(target)
	ch := make(chan interface{})
	go walkPath(ch, sub, source)
	out := false
	var du *docu.Docu
	var output *os.File
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			du = docu.New(mode)
			source = i.(string)
			paths, err = du.Parse(source, nil)
		}
		if err != nil {
			break
		}
		for _, key := range paths {
			output, err = fs.Create(key, lang, ext)
			if err != nil {
				break
			}
			if target == "" {
				if out {
					io.WriteString(output, sp)
				}
				out = true
			}
			err = godoc(output, key, du.FileSet, du.MergePackageFiles(key))
			if target != "" {
				output.Close()
			}
			if err != nil {
				break
			}
		}
		if err != nil {
			break
		}
		ch <- nil
	}
	close(ch)
	if err != nil {
		log.Fatal(err)
	}

}

// walkPath 通道类型约定:
//   nil        结束
//   string     待处理局对路径
//   error      处理错误
func walkPath(ch chan interface{}, sub bool, source string) {
	if strings.HasSuffix(source, ".go") {
		ch <- source
		<-ch
		ch <- nil
		return
	}
	docu.WalkPath(source, func(path string, _ os.FileInfo, err error) error {
		if err == nil {
			ch <- path
		} else {
			ch <- err
		}
		i := <-ch

		if i != nil || !sub || err != nil {
			return io.EOF
		}
		return nil
	})
	ch <- nil
}

func diffMode(mode docu.Mode, sub bool, source, target string) {
	var err error
	var ok bool
	var paths, tpaths []string
	var du, tu *docu.Docu

	output := os.Stdout
	// 如果目录结构不同, 不进行文档对比
	if sub {
		d1 := diffTree(mode, source, target)
		if d1 {
			fmt.Fprintf(output, sp)
		}
		d2 := diffTree(mode, target, source)
		if d1 || d2 {
			return
		}
	}

	ch := make(chan interface{})
	go walkPath(ch, sub, source)

	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			du = docu.New(mode)
			paths, err = du.Parse(i.(string), nil)
		}
		if err == nil {
			if len(paths) == 0 {
				// BUG: 有可能一个为空目录, 另一个非空
				ch <- nil
				continue
			}
			tu = docu.New(mode)
			tpaths, err = tu.Parse( // 处理多包
				filepath.Join(target, strings.Split(paths[0], "::")[0]), nil)
		}

		if err != nil {
			break
		}

		ok, err = docu.SameText(output, "packages "+strings.Join(paths, ","),
			"packages "+strings.Join(tpaths, ","))
		if err != nil {
			break
		}

		if !ok {
			ch <- nil
			io.WriteString(output, sp)
			continue
		}

		for _, key := range paths {
			ok, err = docu.Same(output, du.MergePackageFiles(key), tu.MergePackageFiles(key))
			if !ok && err == nil {
				_, err = io.WriteString(output, "FROM: package ")
				if err == nil {
					_, err = io.WriteString(output, key)
				}
				if err == nil {
					_, err = io.WriteString(output, sp)
				}
			}
			if err != nil {
				break
			}
		}

		if err != nil {
			break
		}
		ch <- nil
	}
	close(ch)
	if err != nil {
		log.Fatal(err)
	}
}

// diffTree 比较目录结构是否相同
func diffTree(mode docu.Mode, source, target string) (diff bool) {
	var ok bool
	var err error
	var fi os.FileInfo
	ch := make(chan interface{})
	go walkPath(ch, true, source)
	pos := len(source)
	if strings.LastIndexAny(source, `\/`)+1 != pos {
		pos++
	}
	output := os.Stdout
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			source = i.(string)
			source = source[pos:]
		}
		if err != nil {
			break
		}
		fi, err = os.Stat(filepath.Join(target, source))
		if os.IsNotExist(err) {
			diff = true
			_, err = fmt.Fprintf(output, "TEXT:\n    path %s\nDIFF:\n    none\n\n", source)
		} else if err == nil && !fi.IsDir() {
			diff = true
			_, err = fmt.Fprintf(output, "TEXT:\n    path %s\nDIFF:\n    is file\n\n", source)
		}
		if err != nil {
			break
		}
		ch <- nil
	}
	close(ch)
	if err != nil {
		log.Fatal(err)
	}
	return
}
