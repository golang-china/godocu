// +build go1.5

package main

import (
	"bytes"
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
	"text/template"

	"github.com/golang-china/godocu/docu"
)

const mode = ast.FilterFuncDuplicates |
	ast.FilterUnassociatedComments | ast.FilterImportDuplicates

const usage = `Usage:

    godocu command [arguments] source [target]

The commands are:

  diff    compare the source and target, all difference output
  first   compare the source and target, the first difference output
  tree    compare different directory structure of the source and target
  code    prints a formatted string to target as Go source code
  plain   prints plain text documentation to target as godoc
  tmpl    prints documentation from template
  list    generate godocu style documents list
  merge   merge source doc to target
  replace replace the target untranslated section in source translated section

The source are:

  package import path or absolute path
  the path to a Go source file

The target are:

  the directory as an absolute base path for compare or prints

The arguments are:

  -file string
      template file for tmpl
  -gopath string
      specifies GOPATH (default $GOPATH)
  -goroot string
      specifies GOROOT (default $GOROOT)
  -lang string
      the lang pattern for the output file, form like en or zh_CN
  -p string
      package filtering, "package"|"main"|"test" (default "package")
  -u
      show unexported symbols as well as exported
`

func flagUsage(err string) {
	fmt.Fprintln(os.Stderr, usage)
	if err != "" {
		log.Fatal(err)
	}
	os.Exit(2)
}

func flagParse() (command, source, target, lib, lang, file string, u bool) {
	var gopath string
	flag.StringVar(&file, "file", "", "")
	flag.StringVar(&docu.GOROOT, "goroot", docu.GOROOT, "")
	flag.StringVar(&gopath, "gopath", os.Getenv("GOPATH"), "")
	flag.StringVar(&lang, "lang", "", "")
	flag.StringVar(&lib, "p", "package", "")
	flag.BoolVar(&u, "u", false, "")

	if len(os.Args) < 3 {
		flagUsage("")
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

	err := flag.CommandLine.Parse(args)
	if err != nil {
		flagUsage(err.Error())
	}
	const libs = "package test main "
	if pos := strings.Index(libs, lib); pos == -1 || libs[pos+len(lib)] != ' ' {
		flagUsage("-p must be one of package,test,main. but got" + lib)
	}

	args = flag.Args()

	if len(args) == 0 || len(args) > 2 {
		flagUsage("")
	}
	command = os.Args[1]

	source = args[0]
	if len(args) == 2 {
		target = args[1]
	}

	docu.GOROOT, err = filepath.Abs(docu.GOROOT)
	if err != nil {
		flagUsage("invalid goroot: " + err.Error())
	}

	if gopath != os.Getenv("GOPATH") {
		docu.GOPATHS = filepath.SplitList(gopath)
		for i, path := range docu.GOPATHS {
			docu.GOPATHS[i], err = filepath.Abs(path)
			if err != nil {
				flagUsage("invalid gopath: " + err.Error())
			}
		}
	}
	lang = docu.LangNormal(lang)

	return
}

// 多文档输出分割线
var sp = "\n\n" + strings.Repeat("/", 80) + "\n\n"

func skipOSArch(f func(string) bool) func(string) bool {
	return func(name string) bool {
		return f(name) && !docu.IsOSArchFileEx(name, "linux", "amd64")
	}
}

func genNameFilter(lib, lang string) func(string) bool {
	if lang == "" {
		switch lib {
		case "package":
			return skipOSArch(docu.PackageFilter)
		case "test":
			return skipOSArch(docu.TestFilter)
		case "main":
			return skipOSArch(docu.MainFilter)
		}
		panic("BUG")
	}
	switch lib {
	case "package":
		return skipOSArch(docu.GenNameFilter("doc_" + lang + ".go"))
	case "test":
		return skipOSArch(docu.GenNameFilter("test_" + lang + ".go"))
	case "main":
		return skipOSArch(docu.GenNameFilter("main_" + lang + ".go"))
	}
	panic("BUG")
}

func genFileName(lib, lang, ext string) string {
	if lang == "" {
		return ""
	}
	switch lib {
	case "package":
		return "doc_" + lang + ext
	case "test":
		return "test_" + lang + ext
	case "main":
		return "main_" + lang + ext
	}
	panic("BUG")
}

func createFile(path, name string) (file *os.File, err error) {
	if name == "" || path == "" {
		return os.Stdout, nil
	}
	if err = os.MkdirAll(path, 0777); err == nil {
		file, err = os.OpenFile(filepath.Join(path, name),
			os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	}
	return
}

func main() {
	const cmds = "code plain tmpl list diff first tree merge replace "
	var err error
	var info os.FileInfo
	command, source, target, lib, lang, file, u := flagParse()

	pos := strings.Index(cmds, command)
	if pos == -1 || cmds[pos+len(command)] != ' ' || target == "" && pos > 20 {

		fmt.Fprintln(os.Stderr, usage)
		log.Fatal("invalid command or target")
	}

	sub := strings.HasSuffix(source, "...")
	if sub {
		source = source[:len(source)-3]
	}

	if source == "" {
		source = docu.GOROOT + docu.SrcElem[:4]
	} else {
		source = docu.Abs(source)
	}

	if info, err = os.Stat(source); err != nil {
		flagUsage(err.Error())
	} else if command == "tree" && !info.IsDir() {
		flagUsage("source must be existing directory")
	}

	// 计算导入路径的偏移量
	offset := posForImport(source)
	if offset == -1 {
		flagUsage("invalid source: " + source)
	} else if target == "--" {
		// 同目录输出
		target = source[:offset]
	} else {
		target = docu.Abs(target)
	}
	if command == "tree" {
		sub = true
		if info, err = os.Stat(target); err != nil || !info.IsDir() {
			flagUsage("target must be existing directory")
		}
	}

	ch := make(chan interface{})
	go walkPath(ch, sub, source)

	switch command {
	case "code", "plain":
		err = showMode(command, ch, offset, target, lib, lang, u)
	case "tree":
		// 对比目录结构
		var d1 bool
		prefix := fmt.Sprintf("source: %s\ntarget: %s\n\nsource target path\n",
			source, target)

		d1, err = treeMode(prefix, "  path  none ", "  path  file ", ch, source, target)
		if err != nil {
			break
		}
		close(ch)

		if d1 {
			prefix = ""
		}

		if offset < len(source) {
			target = filepath.Join(target, source[offset:])
			source = source[:offset]
		}

		ch = make(chan interface{})
		go walkPath(ch, sub, target)
		_, err = treeMode(prefix, "  none  path ", "  file  path ", ch, target, source)
	case "first", "diff":
		err = diffMode(command, ch, offset, target, lib, lang, u)
	case "merge":
		err = mergeMode(ch, offset, target, lib, lang)
	case "replace":
		err = replaceMode(ch, offset, target, lib, lang)
	case "list":
		err = listMode(ch, offset, target, lib, lang, u)
	case "tmpl":
		tpl := template.New("Godocu").Funcs(docu.FuncsMap)
		if file != "" {
			tpl, err = tpl.ParseFiles(file)
		} else {
			tpl, err = tpl.Parse(docu.DefaultTemplate)
		}

		if err != nil {
			<-ch
			break
		}
		err = tmplMode(tpl, ch, offset, target, lib, lang, u)
	}

	close(ch)
	if err != nil {
		log.Fatal(err)
	}
}

// 模板
func tmplMode(tmpl *template.Template, ch chan interface{},
	offset int, target, lib, lang string, u bool) (err error) {

	var buf bytes.Buffer

	var ok bool
	var key, source, dst string
	var paths []string
	var output *os.File

	out := false
	du := docu.NewData()
	du.Docu = docu.New()
	du.Docu.Filter = genNameFilter(lib, "")

	tu := docu.New()
	if target != "" {
		tu.Filter = genNameFilter(lib, lang)
	}

	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
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

		key = paths[0]

		if !u {
			if target != "" {
				// 计算目标路径, 第一个可能是单文件
				if dst == "" && strings.HasSuffix(source, ".go") {
					source = filepath.Dir(source)
				}
				dst = filepath.Join(target, source[offset:])

				// 以目标过滤源
				paths, _ = tu.Parse(dst, nil)
				dis := tu.MergePackageFiles(key)
				if dis != nil && paths[0] == key {
					du.SetFilter(docu.SortDecl(dis.Decls).Filter)
				} else {
					du.SetFilter(docu.ExportedFileFilter)
				}
			} else {
				du.SetFilter(docu.ExportedFileFilter)
			}
		}

		buf.Truncate(0)

		du.Key = key
		if err = tmpl.Execute(&buf, du); err != nil {
			break
		}

		if du.Ext == "" {
			continue
		}

		output, err = createFile(dst, genFileName(lib, lang, du.Ext))
		if err != nil {
			break
		}

		if out && target == "" {
			_, err = os.Stdout.WriteString(sp)
		}
		out = true
		if err == nil {
			_, err = output.Write(buf.Bytes())
		}
		if output != os.Stdout {
			output.Close()
		}

		if err != nil {
			break
		}

		ch <- nil
	}
	return
}

// walkPath 通道类型约定:
//   nil        结束
//   string     待处理绝对路径
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

func showMode(command string, ch chan interface{},
	offset int, target, lib, lang string, u bool) (err error) {

	var ok bool
	var key, source, dst string
	var paths []string

	output := os.Stdout
	ext := ".go"
	docgo := docu.DocGo
	if command == "plain" {
		docgo = docu.Godoc
		ext = ".text"
	}

	out := false
	du := docu.New()
	du.Filter = genNameFilter(lib, "")

	tu := docu.New()
	if target != "" {
		tu.Filter = genNameFilter(lib, lang)
	}

	fname := genFileName(lib, lang, ext)

	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
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

		key = paths[0]
		file := du.MergePackageFiles(key)
		file.Unresolved = nil
		if target != "" {
			// 计算目标路径, 第一个可能是单文件
			if dst == "" && strings.HasSuffix(source, ".go") {
				source = filepath.Dir(source)
			}
			dst = filepath.Join(target, source[offset:])
		}

		if !u {
			if target != "" {
				// 以目标过滤源
				paths, _ = tu.Parse(dst, nil)
				dis := tu.MergePackageFiles(key)
				if dis != nil && paths[0] == key {
					docu.SortDecl(dis.Decls).Filter(file)
				} else {
					docu.ExportedFileFilter(file)
				}

				// 自动提取第一个 lang, 只是为了过滤
				if dis != nil && lang == "" {
					lang = tu.NormalLang(key)
					tu.Filter = genNameFilter(lib, tu.NormalLang(key))
					lang = "."
				}

			} else {
				docu.ExportedFileFilter(file)
			}
		}
		if target != "" && lang != "" && lang != "." {
			output, err = createFile(dst, fname)
		}
		if err != nil {
			break
		}

		if out && target == "" {
			_, err = os.Stdout.WriteString(sp)
		}
		out = true
		if err == nil {
			err = docgo(output, key, du.FileSet, file)
		}

		if output != os.Stdout {
			output.Close()
		}

		if err != nil {
			break
		}

		ch <- nil
	}
	return
}

func diffMode(command string, ch chan interface{},
	offset int, target, lib, lang string, u bool) (err error) {

	var ok, diff bool
	var key, source string
	var paths []string
	var output *os.File

	fileDiff := docu.FirstDiff
	if command == "diff" {
		fileDiff = docu.Diff
	}
	du, tu := docu.New(), docu.New()
	du.Filter = genNameFilter(lib, "")
	tu.Filter = genNameFilter(lib, lang)

	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			source = i.(string)
			paths, err = du.Parse(source, nil)
		}
		if err != nil {
			break
		}

		// 只对比相同的包. source 不能有错, target 的错误被忽略.
		if len(paths) != 0 {
			key = paths[0]
			if strings.HasSuffix(source, ".go") {
				source = filepath.Dir(source)
			}
			paths, err = tu.Parse(filepath.Join(target, source[offset:]), nil)
		}

		if len(paths) == 0 || err != nil {
			err = nil
			ch <- nil
			continue
		}

		diff, err = docu.TextDiff(output, "package "+key, "package "+paths[0])

		if err != nil {
			break
		}

		if diff {
			ch <- nil
			io.WriteString(output, sp)
			continue
		}

		src, dis := du.MergePackageFiles(key), tu.MergePackageFiles(key)
		// 自动提取第一个 lang, 只是为了过滤
		if dis != nil && lang == "" {
			lang = tu.NormalLang(key)
			tu.Filter = genNameFilter(lib, lang)
			if lang == "" {
				lang = "."
			}
		}

		if lang != "." {
			docu.SortDecl(dis.Decls).Filter(src)
		} else if !u {
			docu.ExportedFileFilter(src)
			docu.ExportedFileFilter(dis)
		}

		diff, err = fileDiff(output, src, dis)
		if diff && err == nil {
			_, err = io.WriteString(output, "FROM: package "+key)
		}

		if err != nil {
			break
		}
		ch <- nil
	}
	return
}

// posForImport 计算 import paths 开始的偏移量
func posForImport(s string) (pos int) {
	if strings.HasSuffix(s, ".go") {
		s = filepath.Dir(s)
	}
	if strings.HasSuffix(s, docu.SrcElem[:4]) {
		return len(s) + 1
	}

	pos = strings.Index(s, docu.SrcElem)
	if pos != -1 {
		pos += len(docu.SrcElem)
		return
	}
	for _, wh := range docu.Warehouse {
		pos = strings.Index(s, wh.Host)
		if pos == -1 {
			continue
		}
		if s[pos-1] == os.PathSeparator && s[pos+len(wh.Host)] == os.PathSeparator {
			return pos
		}
	}

	return -1
}

func treeMode(prefix, prenone, prefile string, ch chan interface{}, source, target string) (diff bool, err error) {
	var fi os.FileInfo
	pos := posForImport(source)
	if pos == -1 {
		<-ch
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
		source = source[pos:]

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

func mergeMode(ch chan interface{},
	offset int, target, lib, lang string) (err error) {

	var ok bool
	var key, source, dst string
	var paths []string
	var output *os.File

	out := false
	du := docu.New()
	du.Filter = genNameFilter(lib, "")

	tu := docu.New()
	tu.Filter = genNameFilter(lib, lang)

	fname := genFileName(lib, lang, ".go")

	// 以 target 限制为过滤条件, 因此允许所有
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
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

		key = paths[0]

		// 计算目标路径, 第一个可能是单文件
		if dst == "" && strings.HasSuffix(source, ".go") {
			source = filepath.Dir(source)
		}
		dst = filepath.Join(target, source[offset:])

		paths, _ = tu.Parse(filepath.Join(dst, fname), nil)
		dis := tu.MergePackageFiles(key)

		if dis == nil || len(paths) == 0 || paths[0] != key {
			ch <- nil
			continue
		}

		// 自动提取第一个 lang, 只是为了过滤
		if lang == "" {
			lang = tu.NormalLang(key)
			if lang == "" {
				err = errors.New("missing argument lang")
				break
			}
			tu.Filter = genNameFilter(lib, lang)
			output = os.Stdout
			fname = genFileName(lib, lang, ".go")
			lang = "."
		}

		src := du.MergePackageFiles(key)

		// src 为输出结果, 用目标过滤源
		docu.SortDecl(dis.Decls).Filter(src)

		if !docu.EqualComment(src.Doc, dis.Doc) {
			docu.MergeDoc(dis.Doc, src.Doc)
		}

		docu.MergeDeclsDoc(dis.Decls, src.Decls)

		if lang != "." {
			output, err = createFile(dst, fname)
		}

		if err != nil {
			break
		}

		if lang == "." {
			if out {
				_, err = os.Stdout.WriteString(sp)
			}
			out = true
		}

		if err == nil {
			err = docu.DocGo(output, key, du.FileSet, src)
		}
		if output != os.Stdout {
			output.Close()
		}

		if err != nil {
			break
		}
		ch <- nil
	}
	return
}

func replaceMode(ch chan interface{},
	offset int, target, lib, lang string) (err error) {

	var ok bool
	var key, source, dst string
	var paths []string

	output := os.Stdout

	out := false
	du := docu.New()
	du.Filter = genNameFilter(lib, lang)

	tu := docu.New()
	tu.Filter = du.Filter

	fname := genFileName(lib, lang, ".go")

	// 以 target 限制为过滤条件, 因此允许所有
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
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

		key = paths[0]

		// 计算目标路径, 第一个可能是单文件
		if dst == "" && strings.HasSuffix(source, ".go") {
			source = filepath.Dir(source)
		}
		dst = filepath.Join(target, source[offset:])

		paths, _ = tu.Parse(filepath.Join(dst, fname), nil)
		dis := tu.MergePackageFiles(key)
		if dis == nil || len(paths) == 0 || paths[0] != key {
			ch <- nil
			continue
		}

		src := du.MergePackageFiles(key)
		if !docu.IsGodocuFile(src) || !docu.IsGodocuFile(dis) {
			err = errors.New("source and target must be GodocuStyle documents")
			break
		}

		// 自动提取第一个 lang, 只是为了过滤
		if lang == "" {
			lang = du.NormalLang(key)
			if lang == "" || lang != tu.NormalLang(key) {
				err = errors.New("missing argument lang")
				break
			}
			du.Filter = genNameFilter(lib, lang)
			tu.Filter = du.Filter
			fname = genFileName(lib, lang, ".go")
			lang = "."
		}

		docu.Replace(dis, src)

		if len(dis.Imports) == 0 {
			dis.Imports = src.Imports
		}

		if lang != "." {
			output, err = createFile(dst, fname)
		}

		if err != nil {
			break
		}

		if lang == "." {
			if out {
				_, err = os.Stdout.WriteString(sp)
			}
			out = true
		}

		if err == nil {
			err = docu.DocGo(output, key, tu.FileSet, dis)
		}

		if output != os.Stdout {
			output.Close()
		}

		if err != nil {
			break
		}
		ch <- nil
	}
	return
}

func listMode(ch chan interface{}, offset int, target, lib, lang string, u bool) (err error) {
	var ok bool
	var source string
	var paths []string
	var list docu.List
	var bs []byte

	du := docu.New()
	du.Filter = genNameFilter(lib, lang)
	list.Filename = genFileName(lib, lang, ".go")

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
		// 提取现有属性
		if bs != nil {
			err = json.Unmarshal(bs, &list)
			if err != nil {
				return err
			}
		}

		if list.Readme == "" {
			list.Readme = docu.LookReadme(filepath.Dir(target))
		} else {
			info, err := os.Lstat(filepath.Join(filepath.Dir(target), list.Readme))
			if err != nil || info.IsDir() {
				return errors.New("invalid readme file: " + list.Readme)
			}
		}
		list.Package = nil
		if lang == "" {
			lang = docu.LangOf(list.Filename)
			list.Filename = genFileName(lib, lang, ".go")
			du.Filter = genNameFilter(lib, lang)
			lang = "."
		}
	}

	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			source = i.(string)
			paths, err = du.Parse(source, nil)
		}
		if err != nil {
			break
		}

		if len(paths) == 0 {
			// 简单预测官方包
			if list.Repo == "" {
				if filepath.Base(source) == "archive" {
					list.Repo = "github.com/golang/go"
				}
			}
			ch <- nil
			continue
		}
		key := paths[0]
		// 自动提取第一个 lang, 只是为了过滤
		if lang == "" {
			lang = du.NormalLang(key)
			list.Filename = genFileName(lib, lang, ".go")
			du.Filter = genNameFilter(lib, lang)
			lang = "."
		}

		file := du.MergePackageFiles(key)

		info := docu.Info{
			Synopsis: doc.Synopsis(file.Doc.Text()),
			Progress: docu.TranslationProgress(file),
			Readme:   docu.LookReadme(source),
			Import:   source[offset:],
		}

		list.Package = append(list.Package, info)

		if list.Repo == "" {
			// 官方包引向 github, 其它引向 "localhost"
			imp := info.Import
			if strings.HasPrefix(imp, "golang.org/x") {
				list.Repo = "github.com/golang/tools"
			} else if pos := strings.IndexByte(imp, '/'); pos != -1 {
				imp = imp[:pos]
			}
			if list.Repo == "" && strings.IndexByte(imp, '.') != -1 {
				for _, wh := range docu.Warehouse {
					if wh.Host != imp {
						continue
					}
					pos := 1
					for i := 0; i < wh.Part; i++ {
						end := strings.IndexByte(info.Import[pos:], '/')
						if end == -1 {
							break
						}
						pos += end + 1
					}
					list.Repo = info.Import[:pos-1]
					break
				}
			}
			if list.Repo == "" {
				list.Repo = "localhost"
			}
		}

		ch <- nil
	}

	if err == nil {
		bs, err = json.MarshalIndent(list, "", "    ")
	}
	if err == nil {
		if target == "" || lang == "." {
			_, err = os.Stdout.Write(bs)
			return
		}
		var output *os.File
		output, err = os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err == nil {
			_, err = output.Write(bs)
			output.Close()
		}
	}
	return
}
