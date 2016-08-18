package docu

import (
	"errors"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
)

// SrcElem 本地文件系统中的 "/src/".
const SrcElem = string(os.PathSeparator) + "src" + string(os.PathSeparator)

// IsNormalName 返回 name 是否符合 Docu 命名风格.
// name 必须具有扩展名.
func IsNormalName(name string) bool {
	pos := strings.LastIndexByte(name, '.')
	if pos == -1 || pos+1 == len(name) {
		return false
	}
	name = name[:pos]
	ss := strings.SplitN(name, "_", 2)
	if ss[0] != "doc" && ss[0] != "main" && ss[0] != "test" {
		return false
	}
	return len(ss) == 1 || IsNormalLang(ss[1])
}

func IsNormalLang(lang string) bool {
	ss := strings.SplitN(lang, "_", 2)
	if lang = ss[0]; lang == "" {
		return false
	}
	for i := 0; i < len(lang); i++ {
		if lang[i] < 'a' || lang[i] > 'z' {
			return false
		}
	}
	if len(ss) == 1 {
		return true
	}
	if lang = ss[1]; lang == "" {
		return false
	}
	for i := 0; i < len(lang); i++ {
		if lang[i] < 'A' || lang[i] > 'Z' {
			return false
		}
	}
	return true
}

// LangOf 返回 Godocu  命名风格的 name 中的 lang 部分.
// 如果 name 不符合 Godocu 命名风格返回空.
func LangOf(name string) string {
	pos := strings.IndexByte(name, '.')
	if pos == -1 {
		return ""
	}
	name = name[:pos]

	pos = strings.IndexByte(name, '_')
	if pos == -1 {
		return ""
	}
	switch name[:pos] {
	default:
		return ""
	case "doc", "main", "test":
	}
	if !IsNormalLang(name[pos+1:]) {
		return ""
	}
	return name[pos+1:]
}

// LangNormal 对 lang 进行检查并格式化.
// 如果 lang 不符合要求, 返回空字符串.
func LangNormal(lang string) string {
	ss := strings.Split(strings.ToLower(lang), "_")
	lang = ss[0]
	if len(ss) > 2 || lang == "" {
		return ""
	}
	if len(ss) == 2 {
		if ss[1] == "" {
			return ""
		}
		lang += "_" + strings.ToUpper(ss[1])
	}
	for i := 0; i < len(lang); i++ {
		if lang[i] == '_' ||
			lang[i] >= 'a' && lang[i] <= 'z' ||
			lang[i] >= 'A' && lang[i] <= 'Z' {
			continue
		}
		return ""
	}
	return lang
}

// NormalPkgFileName 返回符合 Docu 命名风格的 pkg 所使用的文件名.
// 如果 pkg 不符合  Docu 命名风格返回空字符串.
func NormalPkgFileName(pkg *ast.Package) string {
	if pkg == nil || len(pkg.Files) != 1 {
		return ""
	}

	for abs := range pkg.Files {
		abs = filepath.Base(abs)
		if !strings.HasSuffix(abs, ".go") || !IsNormalName(abs) {
			break
		}
		return abs
	}

	return ""
}

// LookReadme 在 path 下搜寻 readme 文件名. 未找到返回空串.
func LookReadme(path string) string {
	names, err := readDirNames(path)
	if err != nil {
		return ""
	}

	for _, name := range names {
		if !strings.HasSuffix(name, ".go") &&
			strings.HasPrefix(strings.ToLower(name), "readme") {
			info, err := os.Lstat(filepath.Join(path, name))
			if err != nil && !info.IsDir() {
				return name
			}
		}
	}
	return ""
}

// LookImportPath 返回绝对目录路径 abs 中的 import paths 值. 未找到返回 ""
func LookImportPath(abs string) string {
	if abs == "" {
		return ""
	}
	if abs[len(abs)-1] == os.PathSeparator {
		abs = abs[:len(abs)-1]
	}

	if strings.HasSuffix(abs, SrcElem[:4]) {
		return ""
	}

	pos := strings.Index(abs, SrcElem)
	if pos != -1 {
		return filepath.ToSlash(abs[pos+5:])
	}
	for _, wh := range Warehouse {
		pos = strings.Index(abs, wh.Host)
		if pos == -1 {
			continue
		}
		if abs[pos-1] == os.PathSeparator && abs[pos+len(wh.Host)] == os.PathSeparator {
			return filepath.ToSlash(abs[pos:])
		}
	}

	return ""
}

// IsOSArchFile 返回 name 是否为特定平台文件命名
//     name_$(GOOS).*
//     name_$(GOARCH).*
//     name_$(GOOS)_$(GOARCH).*
//     name_$(GOOS)_test.*
//     name_$(GOARCH)_test.*
//     name_$(GOOS)_$(GOARCH)_test.*
func IsOSArchFile(name string) bool {
	if dot := strings.Index(name, "."); dot != -1 {
		name = name[:dot]
	}
	i := strings.IndexByte(name, '_')
	if i < 0 {
		return false
	}

	l := strings.Split(name[i:], "_")
	if n := len(l); n > 0 && l[n-1] == "test" {
		l = l[:n-1]
	}

	if len(l) >= 2 {
		return contains(goosList, l[0]) && contains(goarchList, l[1])
	}
	return contains(goosList, l[0]) || contains(goarchList, l[0])
}

func contains(s, sep string) bool {
	pos := strings.Index(s, sep)
	if pos == -1 {
		return false
	}
	if pos == 0 {
		return s[pos+len(sep)] == ' '
	}
	return s[pos-1] == ' ' && (len(s) == len(sep)+pos ||
		s[pos+len(sep)+1] == ' ')
}

// Target 创建(覆盖)统一风格的本地文件.
type Target string

// Create 在 Target 目录建立 path,lang,ext 对应的文件.
// 参数:
//   path 为 Docu.Parse 返回的 paths 元素
//   lang 写入文件内容所使用的语言.
//   ext  文件扩展名, 允许为空
// 最终生成的文件名可能是:
//   doc_lang.ext
//   main_lang.ext
//   test_lang.ext
func (t Target) Create(path, lang, ext string) (file *os.File, err error) {
	if t == "" {
		return os.Stdout, nil
	}
	if lang != "" {
		if lang = LangNormal(lang); lang == "" {
			err = errors.New("Target.Create: invaild path or lang.")
			return
		}
	}

	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}

	doc := "doc"
	if pos := strings.Index(path, "::"); pos != -1 {
		doc, path = path[pos+2:], path[:pos]
	}
	if lang != "" {
		doc += "_" + lang
	}
	if ext != "" {
		if ext[0] != '.' {
			doc += "."
		}
		doc += strings.ToLower(ext)
	}

	path = filepath.Join(string(t), path, doc)
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err == nil {
		file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	}
	return
}

// NormalPath 如果 Target 下 path 子目录的扩展名为 ext 文件都符合 Docu 命名风格,
// 返回绝对路径, 否则返回空字符串.
// 参数:
//   path 为 Docu.Parse 返回的 paths 元素.
func (t Target) NormalPath(path, lang, ext string) string {
	if t == "" {
		return ""
	}

	pos := strings.Index(path, "::")
	if pos != -1 {
		path = path[:pos]
	}

	path = filepath.Join(string(t), path)
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return ""
	}
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	lang = LangNormal(lang)
	if lang != "" {
		lang = "_" + lang + ext
	}
	find := false
	for _, name := range names {
		if ext != "" && !strings.HasSuffix(name, ext) {
			continue
		}
		// 必须都符合规范
		if !IsNormalName(name) {
			return ""
		}
		// 特定 lang 也要有
		if lang != "" && !strings.HasSuffix(name, lang) {
			continue
		}
		find = true

	}
	if !find {
		return ""
	}
	return path
}
