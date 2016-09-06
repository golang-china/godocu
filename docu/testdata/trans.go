// package comments trans
package testdata

// 超 80 列折行. 超 80 列折行. 超 80 列折行. 超 80 列折行. 超 80 列折行. 超 80 列折行.
const (
	_ = iota
	// ok
	Ok // is ok
	// 普通文件
	TypeReg  = '0' // regular file // 普通文件
	TypeLink = '1' // hard link // 硬链接
	// 行间的制表符		被保留
	Link1 = '1'      // 1
	Link2 = "333333" // 3
)

// Interface 接口用于标识运行时错误。
type Interface interface {
	error
	// RuntimeError 是一个无操作函数，它只用于区分是运行时错误还是一般错误：
	// 若一个类型拥有 RuntimeError 方法，它就是运行时错误。
	RuntimeError()
}

// comments
type Template struct { // 该尾注释被丢弃
}

type (
	// CSS用于包装匹配如下任一条的已知安全的内容：
	//
	//     1. 此行应该保持原样，不折行.	此行应该保持原样，不折行. tab 不转空格	tab 不转空格
	CSS string // CSS encapsulates // CSS3 样式表

	// 超 80 列折行. 超 80 列折行. 超 80 列折行. 超 80 列折行. 超 80 列折行. 超 80 列折行.
	Wrap struct { // 该尾注释被丢弃
		// 此翻译把 Name 和 Err 分开写了
		Name string // Name // 结构差异
		Err  string
	}

	// Error 描述在模板转义时出现的错误。
	Error struct {
		// ErrorCode
		ErrorCode ErrorCode
		Name      string
		Line      int
		// Description 是人类可读的问题描述。
		Description string
		Call        func(string) string
	}
)
