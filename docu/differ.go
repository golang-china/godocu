package docu

import (
	"fmt"
	"strings"
	"unicode/utf8"
	"unsafe"
)

func str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func bytes2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// Differ 实现 io.Writer, 用于文本对比, 遇到第一个不同返回带行号的错误.
// 文本尾部的 "\n" 不参与对比.
//
// Differ 的算法比较简单, 最初目的是为了方便测试.
type Differ struct {
	name, want        string
	offset, pos, line int
}

// NewDiffer 返回一个 Differ 实例. name 用于错误提示, want 用于对比写入数据.
func NewDiffer(name string, want string) *Differ {
	return &Differ{name, strings.TrimRight(want, "\n"), 0, 0, 0}
}

func (d *Differ) error(s string, i int) error {
	want := d.want[d.offset:]
	pos := strings.IndexByte(want, '\n')
	if pos != -1 {
		want = want[:pos]
	}
	pos = strings.LastIndexByte(s[:i], '\n')
	if pos != -1 {
		s = s[pos+1:]
	} else {
		pos = strings.LastIndexByte(d.want[:d.pos], '\n')
		s = d.want[pos+1:d.pos] + s
	}
	pos = strings.IndexByte(s, '\n')
	if pos != -1 {
		s = s[:pos]
	}

	if d.offset >= len(d.want) {
		return fmt.Errorf("%s:%d:\nWANT: EOF\nDIFF: %q", d.name, d.line+1,
			s)
	}
	return fmt.Errorf("%s:%d:\nWANT: %q\nDIFF: %q", d.name, d.line+1,
		want, s)
}

func (d *Differ) last(s string) (i int, err error) {
	for i = 0; i < len(s); i++ {
		if s[i] != '\n' {
			err = d.error(s, i)
			break
		}
	}
	return
}

// Pos 返回当前的行号和该行第一个字符所在偏移量
func (d *Differ) Pos() (line, offset int) {
	return d.line + 1, d.offset
}

// Write 接收并对比 p, 返回有多少字节的数据一致和不一致的错误信息.
func (d *Differ) Write(p []byte) (n int, err error) {
	return d.WriteString(bytes2str(p))
}

func (d *Differ) WriteString(s string) (n int, err error) {
	for len(s) == 0 {
		return
	}

	offset := d.offset
	max := len(d.want) - d.pos
	for i, r := range s {
		if i >= max {
			return d.last(s[i:])
		}
		c, _ := utf8.DecodeRuneInString(d.want[d.pos+i:])
		if r != c {
			n, err = i, d.error(s, i)
			return
		}
		if r == '\n' {
			d.offset = offset + i + 1
			d.line++
		}
	}
	n = len(s)
	d.pos += n
	return
}
