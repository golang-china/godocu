package docu

// List 表示在同一个 repo 下全部包文档信息.
type List struct {
	// Repo 是原源代码所在托管 git 仓库地址.
	// 如果无法识别值为 "localhost"
	Repo string

	// Readme 该 list 或 Repo 的 readme 文件, 自动提取.
	Readme string `json:",omitempty"`

	// 文档文件名
	Filename string
	// Ext 表示除 "go" 格式文档之外的扩展名.
	// 例如: "md text"
	// 该值由使用者手工设置, Godocu 只是保留它.
	Ext string `json:",omitempty"`

	// Subdir 表示文档文件位于 golist.json 所在目录那个子目录.
	// 该值由使用者手工设置, Godocu 只是保留它.
	Subdir string `json:",omitempty"`

	// Description 该 list 或 Repo 的一句话介绍.
	// 该值由使用者手工设置, Godocu 只是保留它.
	Description string `json:",omitempty"`

	// Golist 表示额外的 golist 文件, 类似友链接, 可以是本目录的或外部的.
	// 该值由使用者手工设置, Godocu 只是保留它.
	Golist []string `json:",omitempty"`

	Package []Info // 所有包的信息
}

// Info 表示单个包文档信息.
type Info struct {
	Import   string // 导入路径
	Synopsis string // 自动提取的一句话包摘要
	// Readme 该包下 readme 文件名, 自动提取.
	Readme   string `json:",omitempty"`
	Progress int    // 翻译完成度
}
