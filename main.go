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
    tree    compare different directory structure of the source and target
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
	}
	if source == "" {
		source = docu.GOROOT + docu.SrcElem[:4]
	} else {
		source = docu.Abs(source)
	}

	_, err = os.Stat(source)

	if target == "" {
		if command == "first" || command == "diff" || command == "merge" ||
			command == "tree" {
			command = "help"
		}
	} else if err == nil {
		if target == "--" {
			target = source
		} else {
			target = docu.Abs(target)
			_, err = os.Stat(target)
		}
	}

	if err != nil {
		command = "help"
	}

	if command == "tree" {
		sub = true
	}

	ch := make(chan interface{})
	go walkPath(ch, sub, source)

	switch command {
	case "code", "plain":
		if target == "--" {
			pos := posForImport(source)
			if pos == -1 {
				<-ch
				err = errors.New("invalid path: " + source)
				break
			}
			if pos < len(source) {
				target = source[:pos]
			} else {
				target = source
			}
		}
		if lang == "" && target != "" {
			err = errors.New("missing lang argument for output")
		} else {
			err = showMode(command, mode, ch, target, lang)
		}
	case "tree":
		// 对比目录结构
		var d1 bool

		prefix := fmt.Sprintf("source: %s\ntarget: %s\n\nsource target path\n",
			source, target)

		d1, err = treeMode(prefix, "  path  none ", "  path  file ", mode, ch, source, target)
		if err != nil {
			break
		}
		close(ch)

		if d1 {
			prefix = ""
		}

		pos := posForImport(source)
		if len(source) > pos {
			target = filepath.Join(target, source[pos:])
			source = source[:pos]
		}

		ch = make(chan interface{})
		go walkPath(ch, sub, target)
		_, err = treeMode(prefix, "  none  path ", "  file  path ", mode, ch, target, source)
	case "first", "diff":
		err = diffMode(command, mode, ch, source, target)
	case "merge":
		err = mergeMode(mode, ch, source, target, lang)
	case "list":
		pos := posForImport(source)
		if pos != -1 {
			err = listMode(ch, pos, target, lang)
		} else {
			err = errors.New("invalid path: " + source)
		}
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

	filter := docu.SuffixFilter("_" + lang + ".go")

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
				if path := fs.NormalPath(key, lang, ext); path != "" {
					// 载入全部目标包
					tu = docu.New(mode)
					tu.Filter = filter
					_, err = tu.Parse(path, nil)
					// 有可能老文件有错误
					if err != nil {
						err = nil
					}
				}
			}

			// 以目标过滤源, 否则无 showUnexported 时进行过滤.
			norfile := tu.MergePackageFiles(key)
			if norfile != nil && tu.NormalLang(key) == lang {
				docu.SortDecl(norfile.Decls).Filter(file)
			} else if !showUn {
				docu.ExportedFileFilter(file)
			}

			output, err = fs.Create(key, lang, ext)

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

	pos := posForImport(source)
	if pos == -1 {
		<-ch
		err = errors.New("invalid path: " + source)
		return
	}

	output := os.Stdout
	fileDiff := docu.FirstDiff
	if command == "diff" {
		fileDiff = docu.Diff
	}
	for i := <-ch; i != nil; i = <-ch {
		if err, ok = i.(error); !ok {
			du = docu.New(mode)
			source = i.(string)
			paths, err = du.Parse(source, nil)
		}
		if err != nil {
			break
		}

		// 只对比相同的包. source 不能有错, target 的错误被忽略.
		if len(paths) != 0 {
			tu = docu.New(mode)
			tpaths, err = tu.Parse(filepath.Join(target, source[pos:]), nil)
		}

		if len(paths) == 0 || len(tpaths) == 0 || err != nil {
			err = nil
			ch <- nil
			continue
		}
		tpath := "packages " + tpaths[0]
		for i := 1; i < len(tpaths); i++ {
			tpath += "," + tpaths[i]
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

// posForImport 计算 import paths 开始的偏移量
func posForImport(s string) (pos int) {
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

func treeMode(prefix, prenone, prefile string, mode docu.Mode, ch chan interface{}, source, target string) (diff bool, err error) {
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

func mergeMode(mode docu.Mode, ch chan interface{}, source, target, lang string) (err error) {
	var spaths, tpaths []string
	var du, tu *docu.Docu
	var fs docu.Target
	var output *os.File

	pos := posForImport(source)
	if pos == -1 {
		<-ch
		err = errors.New("invalid path: " + source)
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

func listMode(ch chan interface{}, pos int, target, lang string) (err error) {
	var ok bool
	var source string
	var paths []string
	var du *docu.Docu
	var list docu.List
	var filter func(string) bool
	var bs []byte

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
			json.Unmarshal(bs, &list)
		}

		if list.Readme == "" {
			list.Readme = docu.LookReadme(target)
		}
		if list.Readme != "" {
			info, err := os.Lstat(filepath.Join(target, list.Readme))
			if err != nil || info.IsDir() {
				return errors.New("invalid readme file: " + list.Readme)
			}
		}
		list.Package = nil
	}

	if lang == "" {
		lang = list.Lang
	} else {
		list.Lang = lang
	}

	if lang != "" {
		filter = docu.SuffixFilter("_" + lang + ".go")
	}

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
			return
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
				return
			}
			filter = docu.SuffixFilter("_" + lang + ".go")
		}

		file := du.MergePackageFiles(key)
		info := docu.Info{
			Synopsis: doc.Synopsis(file.Doc.Text()),
			Progress: docu.TranslationProgress(file),
			Readme:   docu.LookReadme(source),
		}
		// 非 std 包 import paths 需要重新计算
		for i, key := range paths {
			pos := strings.LastIndex(key, "::")
			if i == 0 {
				if pos == -1 {
					info.Import = key
				} else {
					info.Import = key[:pos]
				}
			}
			if pos == -1 {
				continue
			}
			if info.Prefix == "" {
				info.Prefix = key[pos+2:]
			} else {
				info.Prefix = "," + key[pos+2:]
			}
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
			if list.Repo == "" && strings.IndexByte(imp, '.') == -1 {
				list.Repo = "github.com/golang/go"
			} else if list.Repo == "" {
				for _, wh := range docu.Warehouse {
					if !strings.HasPrefix(info.Import, wh.Host) || info.Import[len(wh.Host)] != '/' {
						continue
					}
					repo := info.Import
					for i := 0; i < wh.Part; i++ {
						pos := strings.IndexByte(repo[1:], '/') + 1
						if pos == 0 {
							list.Repo += repo
							break
						}
						list.Repo += repo[:pos]
						repo = repo[pos:]
					}
					break
				}
				if list.Repo == "" {
					list.Repo = "localhost"
				}
			}
		}
		if err != nil {
			return
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
