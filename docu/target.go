package docu

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// SrcElem 本地文件系统中的 "/src/".
const SrcElem = string(os.PathSeparator) + "src" + string(os.PathSeparator)

// Target 创建(覆盖)统一风格的本地文件.
type Target string

// Create 在 Target 目录建立 paths,lang,ext 对应的文件.
// 参数:
//   path 为 Docu.Parse 返回的 paths 元素
//   lang 写入文件内容所使用的语言, 非空.
//   ext  文件扩展名, 允许为空
// 最终生成的文件名可能是:
//   doc_lang.ext
//   main_lang.ext
//   test_lang.ext
func (t Target) Create(path, lang, ext string) (file *os.File, err error) {
	if t == "" {
		return os.Stdout, nil
	}

	if lang == "" || path == "" {
		err = errors.New("Target.Create: empty path or lang not allowed.")
		return
	}
	if ext != "" && ext[0] != '.' {
		ext = "." + ext
	}
	doc := "doc"
	if pos := strings.Index(path, "::"); pos != -1 {
		doc, path = path[pos+2:], path[:pos]
	}
	doc += "_" + strings.ToLower(lang+ext)

	path = filepath.Join(string(t), path, doc)
	err = os.MkdirAll(filepath.Dir(path), 0777)
	if err == nil {
		file, err = os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	}
	return
}

// FromSource 用于来源目录和目标目录不同的情况.
// 通过查找 source 中的 SrcElem 位置计算出目标下的子路径, 然后调用 Create.
func (t Target) FromSource(source, path, lang, ext string) (file *os.File, err error) {
	if t == "" {
		return os.Stdout, nil
	}

	pos := strings.LastIndex(source, SrcElem)
	if pos == -1 || pos+5 == len(source) {
		err = errors.New("Target.FromSource: invaild source: " + source)
		return
	}
	// 通过 source 绝对目录计算目标路径
	return t.Create(filepath.Join(source[pos+5:], path), lang, ext)
}
