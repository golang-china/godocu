package docu

import (
	"bytes"
	"errors"
	"go/ast"
	"io"
	"io/ioutil"
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
	return len(ss) == 2 &&
		(ss[0] == "doc" || ss[0] == "main" || ss[0] == "test") &&
		IsNormalLang(ss[1])
}

var declPackage = []byte("\npackage ")
var plusBuild = []byte("+build")
var linux = []byte("linux")
var slashslash = []byte("//")

func buildForLinux(code []byte) bool {
	pos := bytes.Index(code, declPackage)
	if pos == -1 {
		return bytes.HasPrefix(code, declPackage[1:])
	}
	code = code[:pos+1]

	for len(code) != 0 {
		pos = bytes.IndexByte(code, '\n')
		line := code[:pos]
		code = code[pos+1:]
		if !bytes.HasPrefix(line, slashslash) {
			continue
		}
		line = bytes.TrimSpace(line[2:])
		if !bytes.HasPrefix(line, plusBuild) {
			continue
		}
		line = line[len(plusBuild):]
		pos = bytes.Index(line, linux)
		return pos > 0 && line[pos-1] == ' ' &&
			(pos+len(linux) == len(line) || line[pos+len(linux)] == ' ')
	}
	return true
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
			if err == nil && !info.IsDir() {
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

// OSArchTest 提取并返回 go 文件名 name 中可识别的 goos, goarch, test 部分.
//     name_$(GOOS).*
//     name_$(GOARCH).*
//     name_$(GOOS)_$(GOARCH).*
//     name_$(GOOS)_test.*
//     name_$(GOARCH)_test.*
//     name_$(GOOS)_$(GOARCH)_test.*
func OSArchTest(name string) (goos, goarch string, test bool) {
	if !strings.HasSuffix(name, ".go") {
		return
	}
	name = name[:len(name)-3]
	if len(name) == 0 {
		return
	}
	l := strings.Split(name, "_")[1:]
	n := len(l)
	if n == 0 {
		return
	}
	if l[n-1] == "test" {
		test = true
		n--
		l = l[:n]
	}
	if n == 0 {
		return
	}
	if n >= 2 {
		l, n = l[n-2:], 2
	}

	s := l[n-1]
	if contains(goosList, s) {
		goos = s
	} else if contains(goarchList, s) {
		goarch = s
	}
	if n == 1 || goos == "" && goarch == "" {
		return
	}
	s = l[0]
	if goos == "" && contains(goosList, s) {
		goos, s = s, ""
	}
	if goarch == "" && contains(goarchList, s) {
		goarch = s
	}
	return
}

func contains(s, sep string) bool {
	pos := strings.Index(s, sep)
	if pos == -1 {
		return false
	}
	if len(sep) == len(s) {
		return true
	}
	if pos == 0 {
		return s[pos+len(sep)] == ' '
	}
	if pos+len(sep) == len(s) {
		return s[pos-1] == ' '
	}
	return s[pos-1] == ' ' && s[pos+len(sep)] == ' '
}

func readSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch s := src.(type) {
		case string:
			return []byte(s), nil
		case []byte:
			return s, nil
		case *bytes.Buffer:
			// is io.Reader, but src is already available in []byte form
			if s != nil {
				return s.Bytes(), nil
			}
		case io.Reader:
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, s); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
		return nil, errors.New("invalid source")
	}
	return ioutil.ReadFile(filename)
}
