// +build go1.5

package main

import (
	"errors"
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

const usage = `Usage:

    godocu command [arguments] source [target]

The commands are:

    diff    compare the source and target, all difference output
    first   compare the source and target, the first difference output
    code    prints a formatted string to target as Go source code
    plain   prints plain text documentation to target as godoc
    merge   merge source doc to target

The source are:

    package import path or absolute path
    the path to a Go source file

The target are:

    the directory as an absolute base path for compare or prints

The arguments are:
`

func flagUsage() {
	fmt.Fprintln(os.Stderr, usage)
	flag.PrintDefaults()
	os.Exit(2)
}

func flagParse() (mode docu.Mode, command, source, target, lang string) {
	var gopath string
	var u, cmd, test bool

	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "Go root directory")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "specifies gopath")
	flag.StringVar(&lang, "lang", "", "the lang pattern for the output file, form like en or zh_CN")
	flag.BoolVar(&u, "u", false, "show unexported symbols as well as exported")
	flag.BoolVar(&cmd, "cmd", false, "show symbols with package docs even if package is a command")
	flag.BoolVar(&test, "test", false, "show symbols with package docs even if package is a testing")

	if len(os.Args) < 3 {
		flagUsage()
	}

	flag.CommandLine.Parse(os.Args[2:])

	args := flag.Args()
	if len(args) == 0 || len(args) > 2 {
		flagUsage()
	}
	command = os.Args[1]

	source = args[0]
	if len(args) == 2 {
		target = args[1]
	}

	if gopath != os.Getenv("GOPATH") {
		docu.GOPATHS = filepath.SplitList(gopath)
	}

	if u {
		mode |= docu.ShowUnexported
	}
	if cmd {
		mode |= docu.ShowCMD
	}

	if test {
		mode |= docu.ShowTest
	}

	return
}

// 多文档输出分割线
var sp = "\n\n" + strings.Repeat("/", 80) + "\n\n"

func main() {
	var err error
	mode, command, source, target, lang := flagParse()
	sub := strings.HasSuffix(source, "...")
	if sub {
		source = source[:len(source)-3]
		if source == "" {
			source = docu.GOROOT + docu.SrcElem
		}
	}

	if target == "" &&
		(command == "first" || command == "diff" || command == "merge") {
		command = "help"
	}
	source = docu.Abs(source)
	target = docu.Abs(target)

	ch := make(chan interface{})
	go walkPath(ch, sub, source)
	switch command {
	case "code", "plain":
		err = showMode(command, mode, ch, target, lang)
	case "first", "diff":
		target = docu.Abs(target)
		// 如果目录结构不同, 不进行文档对比
		if sub {
			var d1, d2 bool

			prefix := fmt.Sprintf("source: %s\ntarget: %s\n\nsource target import_path\n",
				source, target)

			d1, err = diffTree(prefix, "  path  none ", "  path  file ", mode, ch, source, target)
			if err != nil {
				break
			}
			close(ch)

			pos := posForImport(source)
			tpath := filepath.Join(target, source[pos:])
			if d1 {
				prefix = ""
			}
			ch = make(chan interface{})
			go walkPath(ch, sub, tpath)
			d2, err = diffTree(prefix, "  none  path ", "  file  path ", mode, ch, tpath, source[:pos])
			if err != nil || d1 || d2 {
				break
			}

			ch = make(chan interface{})
			go walkPath(ch, sub, source)
		}
		err = diffMode(command, mode, ch, source, target)
	case "merge":
		err = mergeMode(mode, ch, source, target, lang)
	default:
		<-ch
		ch <- false
		<-ch
		close(ch)
		flagUsage()
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

func showMode(command string, mode docu.Mode, ch chan interface{}, target, lang string) (err error) {
	var ok bool
	var source string
	var paths []string
	var du, tu *docu.Docu
	var output *os.File

	fs := docu.Target(target)
	ext := ".go"
	docgo := docu.DocGo
	if command == "plain" {
		docgo = docu.Godoc
		ext = ".text"
	}
	// 先不要过滤
	showUn := mode&docu.ShowUnexported != 0
	mode |= docu.ShowUnexported

	out := false
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			du = docu.New(mode)
			source = i.(string)
			paths, err = du.Parse(source, nil)
		}
		if err != nil {
			break
		}

		for i, key := range paths {
			file := du.MergePackageFiles(key)
			if i == 0 && target != "" {
				path := fs.NormalPath(key, lang)
				if path != "" {
					// 载入全部包
					tu = docu.New(mode)
					_, err = tu.Parse(path, nil)
					if err != nil {
						break
					}
				}
			}

			norfile := tu.MergePackageFiles(key)
			if norfile != nil {
				docu.SortDecl(norfile.Decls).Filter(file)
			} else if !showUn {
				docu.ExportedFileFilter(file)
			}
			// 测试一下
			if false {
				output, err = fs.Create(key, lang, ext)
			}
			output = os.Stdout
			if err != nil {
				break
			}
			if target == "" {
				if out {
					io.WriteString(output, sp)
				}
				out = true
			}
			err = docgo(output, key, du.FileSet, file)
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
	return
}

func diffMode(command string, mode docu.Mode, ch chan interface{}, source, target string) (err error) {
	var ok, diff bool
	var paths, tpaths []string
	var du, tu *docu.Docu

	output := os.Stdout
	fileDiff := docu.FirstDiff
	if command == "diff" {
		fileDiff = docu.Diff
	}
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

		diff, err = docu.TextDiff(output, "packages "+strings.Join(paths, ","),
			"packages "+strings.Join(tpaths, ","))
		if err != nil {
			break
		}

		if diff {
			ch <- nil
			io.WriteString(output, sp)
			continue
		}

		for _, key := range paths {
			diff, err = fileDiff(output, du.MergePackageFiles(key), tu.MergePackageFiles(key))
			if diff && err == nil {
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
	return
}

func posForImport(s string) (pos int) {
	pos = strings.Index(s, docu.SrcElem)
	if pos != -1 {
		pos += len(docu.SrcElem)
	} else if strings.HasSuffix(s, docu.SrcElem[:len(docu.SrcElem)-1]) {
		pos = len(s)
	}
	return
}

// diffTree 比较目录结构是否相同
func diffTree(prefix, prenone, prefile string, mode docu.Mode, ch chan interface{}, source, target string) (diff bool, err error) {
	var fi os.FileInfo
	pos := posForImport(source)
	if pos == -1 {
		err = errors.New("invalid path: " + source)
		fmt.Println(prefix)
		return
	}
	if pos == len(source) && source[pos-1] != os.PathSeparator {
		pos++ // 去掉 "/"
	}
	output := os.Stdout
	for i := <-ch; i != nil; i = <-ch {
		err, _ = i.(error)
		if err != nil {
			break
		}

		source = i.(string)
		source = source[pos:] // 计算 import path
		fi, err = os.Stat(filepath.Join(target, source))
		if os.IsNotExist(err) {
			_, err = fmt.Fprintln(output, prefix+prenone, source)
			if err != nil {
				break
			}
			diff, err, prefix = true, nil, ""
		} else if err == nil && !fi.IsDir() { // 虽然不大能
			_, err = fmt.Fprintln(output, prefix+prefile, source)
			if err != nil {
				break
			}
			diff, prefix = true, ""
		}
		if err != nil {
			break
		}
		ch <- nil
	}
	return
}

func mergeMode(mode docu.Mode, ch chan interface{}, source, target, lang string) (err error) {
	var ok, diff bool
	var paths, tpaths []string
	var du, tu *docu.Docu

	// fs := docu.Target(target)

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
		diff, err = docu.TextDiff(os.Stdout, "packages "+strings.Join(paths, ","),
			"packages "+strings.Join(tpaths, ","))

		if err != nil {
			break
		}

		if diff {
			err = errors.New("the difference between source and target")
			break
		}

		for _, key := range paths {
			src := du.MergePackageFiles(key)
			dis := tu.MergePackageFiles(key)
			if len(dis.Imports) == 0 {
				dis.Imports = src.Imports
			}

			if !docu.DiffFormOnly(src.Doc.Text(), dis.Doc.Text()) {
				docu.MergeDoc(src.Doc, dis.Doc)
			}

			docu.MergeDecls(src.Decls, dis.Decls)
			err = docu.DocGo(os.Stdout, key, tu.FileSet, dis)
			if err != nil {
				break
			}
		}

		if err != nil {
			break
		}
		ch <- nil
	}
	return
}
