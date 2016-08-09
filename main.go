// +build go1.5

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"io"
	"io/ioutil"
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
    list    prints godocu style documents list
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

	args := make([]string, len(os.Args[2:]))
	j := 0
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i][0] == '-' && os.Args[i] != "--" {
			args[j] = os.Args[i]
			j++
		}
	}

	for i := 2; i < len(os.Args); i++ {
		if os.Args[i][0] != '-' || os.Args[i] == "--" {
			args[j] = os.Args[i]
			j++
		}
	}

	flag.CommandLine.Parse(args)
	args = flag.Args()

	// flag.CommandLine.Parse(os.Args[2:])
	// args := flag.Args()

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
	lang = docu.LangNormal(lang)

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
	ch := make(chan interface{})
	go walkPath(ch, sub, source)
	switch command {
	case "code", "plain":
		if target == "--" {
			pos := posForImport(source)
			if pos == -1 {
				<-ch
				err = errors.New("invalid source: " + source)
				break
			}
			if pos < len(source) {
				target = source[:pos]
			} else {
				target = source
			}
		}
		err = showMode(command, mode, ch, target, lang)
	case "first", "diff":
		target = docu.Abs(target)
		// 如果目录结构不同, 不进行文档对比
		if sub {
			var d1, d2 bool
			var spath, tpath string

			prefix := fmt.Sprintf("source: %s\ntarget: %s\n\nsource target import_path\n",
				source, target)

			d1, err = diffTree(prefix, "  path  none ", "  path  file ", mode, ch, source, target)
			if err != nil {
				break
			}
			close(ch)

			// 交换 source,target
			pos := posForImport(source)
			if pos < len(source) {
				spath, tpath = filepath.Join(target, source[pos:]), source[:pos]
			} else {
				spath, tpath = target, source
			}

			if d1 {
				prefix = ""
			}
			ch = make(chan interface{})
			go walkPath(ch, sub, spath)
			d2, err = diffTree(prefix, "  none  path ", "  file  path ", mode, ch, spath, tpath)
			if err != nil || d1 || d2 {
				break
			}

			ch = make(chan interface{})
			go walkPath(ch, sub, source)
		}
		err = diffMode(command, mode, ch, source, target)
	case "merge":
		err = mergeMode(mode, ch, source, target, lang)
	case "list":
		err = listMode(ch, target, lang)
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
			lan := lang
			if i == 0 && target != "" {
				if path := fs.NormalPath(key, lang, ext); path != "" {
					// 载入全部包
					tu = docu.New(mode)
					_, err = tu.Parse(path, nil)
					if err != nil {
						break
					}
				}
			}

			// 有目标, 以目标过滤源, 否则无 showUnexported 时进行过滤.
			norfile := tu.MergePackageFiles(key)
			if norfile != nil {
				docu.SortDecl(norfile.Decls).Filter(file)
				lan = tu.NormalLang(key) // 提取目标语言作为文件名参数
			} else if !showUn {
				docu.ExportedFileFilter(file)
			}

			output, err = fs.Create(key, lan, ext)
			lang = lan

			if err == nil && out && target == "" {
				_, err = os.Stdout.WriteString(sp)
			}
			out = true
			if err == nil {
				err = docgo(output, key, du.FileSet, file)
			}
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
	var offset int
	if posForImport(target) == -1 {
		offset = len(target)
		if target[offset-1] != '/' && target[offset-1] != '\\' {
			offset++
		}
	}

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
		if err != nil {
			break
		}
		// 必须两个都有相同的包才进行对比.
		// source 不能有错, 因已经进行了 diffTree 比较, target 的错误被忽略.
		tu = docu.New(mode)
		tpaths, err = tu.Parse( // 处理多包
			filepath.Join(target, strings.Split(paths[0], "::")[0]), nil)

		if len(paths) == 0 || len(tpaths) == 0 {
			err = nil
			ch <- nil
			continue
		}
		tpath := "packages " + tpaths[0][offset:]
		for i := 1; i < len(tpaths); i++ {
			tpath += "," + tpaths[i][offset:]
		}

		diff, err = docu.TextDiff(output, "packages "+strings.Join(paths, ","), tpath)

		if err != nil {
			break
		}

		if diff {
			ch <- nil
			io.WriteString(output, sp)
			continue
		}

		for i, key := range paths {
			diff, err = fileDiff(output, du.MergePackageFiles(key), tu.MergePackageFiles(tpaths[i]))
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
		pos = len(s) + 1
	}
	return
}

// diffTree 比较目录结构是否相同
func diffTree(prefix, prenone, prefile string, mode docu.Mode, ch chan interface{}, source, target string) (diff bool, err error) {
	var fi os.FileInfo
	pos := posForImport(source)
	if pos == -1 {
		err = errors.New("invalid path: " + source)
		return
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
	var spaths, tpaths []string
	var du, tu *docu.Docu
	var fs docu.Target
	var output *os.File

	// source 必须含有 '/src/'
	pos := posForImport(source)
	if pos == -1 {
		<-ch
		err = errors.New("invalid source: " + source)
		return
	}

	if lang == "" {
		fs = docu.Target("")
	} else {
		fs = docu.Target(target)
	}

	// 以 target 限制为过滤条件, 因此允许所有
	for i := <-ch; i != nil; i = <-ch {
		if err, _ = i.(error); err != nil {
			break
		}
		// 先得到目标
		tu = docu.New(docu.ShowUnexported | docu.ShowCMD | docu.ShowTest)
		source = i.(string)
		tpaths, err = tu.Parse(filepath.Join(target, source[pos:]), nil)

		// 计算 mode
		mode = docu.ShowUnexported
		for _, key := range tpaths {
			key = docu.NormalPkgFileName(tu.Package(key))
			if key == "" {
				tpaths = nil
				break
			}
			if strings.HasPrefix(key, "main") {
				mode |= docu.ShowCMD
			} else if strings.HasPrefix(key, "test") {
				mode |= docu.ShowTest
			}
		}
		if len(tpaths) == 0 {
			err = nil
			ch <- nil
			continue
		}

		// 必须有相同的包才进行对比.
		// source 不能有错, target 的错误被忽略.
		du = docu.New(mode)
		spaths, err = du.Parse(source, nil)
		if err != nil {
			break
		}
		if len(spaths) == 0 {
			ch <- nil
			continue
		}

		for _, key := range tpaths {
			// 过滤掉非指定的语言
			lan := tu.NormalLang(key)
			if lang != "" && lang != lan {
				continue
			}

			src := du.MergePackageFiles(key)
			// 有可能 source 发生了变化
			if src == nil {
				err = errors.New("lost package " + key + ", on " + source)
				break
			}

			dis := tu.MergePackageFiles(key)
			// 总是以目标过滤源
			docu.SortDecl(dis.Decls).Filter(src)

			if len(dis.Imports) == 0 {
				dis.Imports = src.Imports
			}

			if src.Doc != nil {
				if dis.Doc == nil {
					dis.Doc = src.Doc
				} else {
					docu.MergeDoc(src.Doc, dis.Doc)
				}
			}

			docu.MergeDecls(src.Decls, dis.Decls)
			output, err = fs.Create(key, lan, ".go")
			if err == nil {
				err = docu.DocGo(output, key, tu.FileSet, dis)
				if lang != "" {
					output.Close()
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

type description struct {
	Repo, Description, Subdir string
}

func listMode(ch chan interface{}, target, lang string) (err error) {
	var ok bool
	var source string
	var paths []string
	var du *docu.Docu
	var list docu.List
	var filter func(string) bool
	var bs []byte

	if lang != "" {
		filter = docu.SuffixFilter("_" + lang + ".go")
	}

	if target != "" {
		info, err := os.Stat(target)
		if err != nil {
			return err
		}
		if info.IsDir() {
			target = filepath.Join(target, "golist.json")
		} else if !strings.HasSuffix(info.Name(), ".json") {
			return errors.New("invalid target")
		}
		bs, _ = ioutil.ReadFile(target)
		// 提取 Description
		if bs != nil {
			var desc description
			if json.Unmarshal(bs, &desc) == nil {
				list.Repo = desc.Repo
				list.Description = desc.Description
				list.Subdir = desc.Subdir
			}
		}
	}

	list.Markdown = true
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			du = docu.New(docu.ShowUnexported | docu.ShowCMD | docu.ShowTest)
			if filter != nil {
				du.Filter = filter
			}
			source = i.(string)
			paths, err = du.Parse(source, nil)
		}
		if err != nil {
			break
		}
		if len(paths) == 0 {
			ch <- nil
			continue
		}
		// 因为 paths 已经排序, 取第一个就好
		key := paths[0]
		if filter == nil {
			lang = du.NormalLang(key)
			if lang == "" {
				err = errors.New("invalid NormalLang in source: " + source)
				break
			}
			filter = docu.SuffixFilter("_" + lang + ".go")
		}
		file := du.MergePackageFiles(key)
		info := docu.Info{
			Synopsis: doc.Synopsis(file.Doc.Text()),
			Progress: docu.TranslationProgress(file),
		}
		// 非 std 包 import paths 需要重新计算
		for _, key := range paths {
			pos := strings.LastIndex(key, "::")
			if pos == -1 {
				info.Prefix = "doc"
				continue
			}
			if info.Prefix == "" {
				info.Prefix = key[pos+2:]
			} else {
				info.Prefix = "," + key[pos+2:]
			}
		}
		list.Info = append(list.Info, info)
		if err != nil {
			break
		}
		ch <- nil
	}
	bs, err = json.MarshalIndent(list, "", "    ")
	if err == nil {
		if target == "" {
			_, err = os.Stdout.Write(bs)
		} else {
			var output *os.File
			output, err = os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
			if err == nil {
				_, err = output.Write(bs)
				output.Close()
			}
		}

	}
	return
}
