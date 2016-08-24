# GoDocu

Godocu 基于 [docu][] 实现的指令行工具, 从 Go 源码生成文档.

功能:

  - 80 列换行, 支持多字节字符
  - 若原注释已经符合 80 列换行, 保持不变.
  - 可提取执行包文档, 测试包文档, 非导出符号文档
  - 遍历目录
  - 生成多种风格文档, Go 源码风格, 文本风格, Markdown 风格, 支持模板
  - 生成文档概要清单
  - 合并生成双语文档
  - 合并两份双语文档中的翻译成果
  - 比较两份文档差异
  - 比较两个包目录结构差异
  - 多平台文档只提取 linux, amd64 的组合

该工具在 Golang 官方包下测试通过, 非官方包请核对输出结果.

文件名命名风格:

Godocu 文档文件名由前缀 "doc"|"main"|"test" 和语言后缀 "_lang" 以及扩展名组成.

扩展名

 - `code`,`merge`,`replace` 指令输出扩展名为 ".go".
 - `plain` 指令扩展名为 ".text".
 - `tmpl` 指令扩展名由模板决定, 比如 ".md"

# Install

```
go get github.com/golang-china/godocu
```

# Usage

```
Usage:

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
```

# source

source 必选, 用于计算源文件绝对路径, 和 import path.
source 可以是 import path 或绝对路径表示的目录或文件.

如果是 import path, 先在 `GOROOT/src`, `GOPATH/src` 下查找并计算出绝对路径.
因 go 源文件绝对路径比较规律, Godocu 通过绝对路径计算出 import path.

在 source 尾部加 `...` 表示遍历子目录.
若 source 值为 `...` 特指全部官方包, 即遍历 `GOROOT/src` 下的所有包.

遍历目录时 Godocu 参照 Go 命名习惯, 忽略 `testdata`, `vendor` 之类的目录.

详情参见相关指令以及 [Example](#example).

# target

target 除 `list` 指令外都表示基础目标路径, 配合 souce 计算出目标路径.

Godocu 要求某个包的目录结构在 source 和 target 下是相同的.

方便起见, target 值为 "--" 表示输出到 source 计算得到的原包目录.

对于 `code`, `plain`,`list`,`tmpl` 指令, target 可选, 缺省输出到 Stdout.

对于 `diff`, `first`, `tree` 指令, target 必选, 结果输出到 Stdout.

对于 `merge',`replace` 指令, target 必选.

*安全起见, 只有显示指定 `lang` 参数, 才会生成或覆盖目标文件, 否则输出到 Stdout*

详情参见相关指令以及 [Example](#example).

# lang

参数 `lang` 指定目标文件名后缀, 格式为 lang 或 lang_ISOCountryCode.
即 lang 部分为小写, ISOCountryCode 部分为大写.

辅助函数 `docu.LangNormal` 提供规范化处理.

方便起见, 未指定 `lang` 时, Godocu 尝试从 target 下首个匹配的包提取 `lang`.

*提示: 不要让多种翻译文档共存同一目录*

详情参见相关指令.

# package_filtering

参数 'p' 用于过滤包, 可选值为 "package","main","test" 之一. 缺省为 "package".

即: Godocu 每次只处理一种类别的包: 库,可执行包或测试包.

# unexported

参数 'u' 允许文档包含顶级非导出声明.

比如 `builtin` 包的声明多是非导出的, 但在文档中是不可或缺的.

Godocu 的非导出优先策略是:

 1. 如果使用了 'u' 参数, 包含全部非导出声明
 2. 如果目标已存在且有 Godocu 风格 ".go" 文件, 其中的非导出声明被保留
 3. 否则不输出非导出声明

该参数对 `merge`, `replace` 指令无效, 因这两个指令的目标必须存在.

# goroot

仅当 source 为 import path 时, 参数 `goroot`,`gopath` 用于计算绝对路径.

# file

参数 'file' 表示外部文件, 目前仅为 `tmpl` 指令指定外部模板文件.

# Merge

指令 `merge` 合并 source 文档到 target 中相同顶级声明的文档之前, 生成翻译文档.

*注意: 输出结果保持 source 的代码结构, merge 在整个工具链中非常重要*

翻译文档生是双(语)文档, 但 merge 不分析文档所用的语言.
翻译文档代码结构可能和原文档不同, 比如:原文档中用了分组, 翻译文档没用.
使用 merge 可保证文档结构风格和原代码一致.

 - source 可以是源码或包文档, 事实上使用源码具有现实意义.
 - target 可以是源码或包文档.
 - 结果总是使用 source 的 import.
 - 如果声明的文档一样, 不追加, 即只有一份文档.
 - source, target 都有尾注释的话, 使用 target 中的尾注释.
 - 指定 `lang` 参数才生成或覆盖 target, 否则仅向 stdout 打印结果.
 - 最终结果 source 中已被删除的声明会被剔除, 新声明会出现.

合并 `builtin` 包文档到 [translations][].

```shell
$ godocu merge builtin /path/to/github.com/golang-china/golangdoc.translations/src
```

遍历所有官方包文档合并到 [translations][].

```shell
$ godocu merge ... /path/to/github.com/golang-china/golangdoc.translations/src
```

# Code

指令 `code` 输出 ".go" 格式单文档.

如果指定了 target 要求参数 `lang` 非空.

输出 `builtin` 包文档, 显然要加参数 'u'

```shell
$ godocu code builtin -u
```

如你所见, Godocu 支持 "-" 开头的参数在任意位置出现.

# Plain

指令 `plain` 输出 ".text" 格式单文档.

如果指定了 target 要求参数 `lang` 非空.

# Tmpl

指令 `tmpl` 支持模板输出, 参数 'file' 指定模板文件, 缺省为内置的 Markdown 模板.

# Tree

指令 `tree` 遍历比较输出 sourec, target 目录结构差异.

该指令总是遍历目录, source 无需加 "..."

遍历比较当前版本 1.6.2 和老版本的差异:

```shell
$ godocu tree ... /usr/local/Cellar/go/1.5.2/libexec/src
```

输出:

```
source: /usr/local/Cellar/go/1.6.2/libexec/src
target: /usr/local/Cellar/go/1.5.2/libexec/src

source target path
  path  none  cmd/compile/internal/mips64
  path  none  cmd/internal/obj/mips
  path  none  cmd/internal/unvendor
  path  none  cmd/internal/unvendor/golang.org
  path  none  cmd/internal/unvendor/golang.org/x
  path  none  cmd/internal/unvendor/golang.org/x/arch
  path  none  cmd/internal/unvendor/golang.org/x/arch/arm
  path  none  cmd/internal/unvendor/golang.org/x/arch/arm/armasm
  path  none  cmd/internal/unvendor/golang.org/x/arch/x86
  path  none  cmd/internal/unvendor/golang.org/x/arch/x86/x86asm
  path  none  cmd/link/internal/mips64
  path  none  cmd/vet/internal
  path  none  cmd/vet/internal/whitelist
  path  none  internal/golang.org
  path  none  internal/golang.org/x
  path  none  internal/golang.org/x/net
  path  none  internal/golang.org/x/net/http2
  path  none  internal/golang.org/x/net/http2/hpack
  path  none  internal/race
  path  none  internal/syscall/windows/sysdll
  path  none  runtime/internal
  path  none  runtime/internal/atomic
  path  none  runtime/internal/sys
  path  none  runtime/msan
  none  path  cmd/internal/rsc.io
  none  path  cmd/internal/rsc.io/arm
  none  path  cmd/internal/rsc.io/arm/armasm
  none  path  cmd/internal/rsc.io/x86
  none  path  cmd/internal/rsc.io/x86/x86asm
  none  path  cmd/vet/whitelist
  none  path  internal/format
```

对比 "cmd" 目录的变化使用:

```shell
$ godocu tree cmd /usr/local/Cellar/go/1.5.2/libexec/src
```

# Diff

指令 `diff` 比较输出 source, target 共有包差异, 指令 `first` 仅输出首个差异.

如果指定了 `lang`, 对 target 进行 `lang` 过滤, 且以 target 的声明为对比条目.

比较 reflect 在当前版本 1.6.2 和老版本的导出声明差异

```shell
$ godocu first reflect /usr/local/Cellar/go/1.5.2/libexec/src
```

输出

```
TEXT:
    func DeepEqual(x, y interface{}) bool
DIFF:
    func DeepEqual(a1, a2 interface{}) bool

FROM: package reflect
```

意思是

```
内容:
    func DeepEqual(x, y interface{}) bool
不同:
    func DeepEqual(a1, a2 interface{}) bool

来自: package reflect
```

比较 os 包在当前版本 1.6.2 和老版本的导出声明差异

```shell
$ godocu diff os /usr/local/Cellar/go/1.5.3/libexec/src
```

输出

```
TEXT:
    Type ProcessState struct{pid int; status syscall.WaitStatus; rusage
    *syscall.Rusage}
DIFF:
    Type ProcessState struct{pid int; status *syscall.Waitmsg}

TEXT:
    func FindProcess(pid int) (*Process, error)
DIFF:
    func FindProcess(pid int) (p *Process, err error)

TEXT:
    func Rename(oldpath, newpath string) error

    Rename renames (moves) oldpath to newpath.
    If newpath already exists, Rename replaces it.
    OS-specific restrictions may apply when oldpath and newpath are in different
    directories.
    If there is an error, it will be of type *LinkError.
DIFF:
    func Rename(oldpath, newpath string) error

    Rename renames (moves) a file. OS-specific restrictions might apply.
    If there is an error, it will be of type *LinkError.

TEXT:
    func (*File) Seek(offset int64, whence int) (ret int64, err error)

    Seek sets the offset for the next Read or Write on file to offset, interpreted
    according to whence: 0 means relative to the origin of the file, 1 means
    relative to the current offset, and 2 means relative to the end.
    It returns the new offset and an error, if any.
    The behavior of Seek on a file opened with O_APPEND is not specified.
DIFF:
    func (*File) Seek(offset int64, whence int) (ret int64, err error)

    Seek sets the offset for the next Read or Write on file to offset, interpreted
    according to whence: 0 means relative to the origin of the file, 1 means
    relative to the current offset, and 2 means relative to the end.
    It returns the new offset and an error, if any.

FROM: package os
```

可以看到结构体和注释有些区别.

Docu 提供了值其实一样, 只是排版格式发生变化的对比, Godocu 只简单比较值

# List

list 指令以 JSON 格式输出 Godocu 风格文档清单.

如果 `lang` 为空, list 尝试自动提取 `lang`.

target:

 - 如果 target 为空, 输出到 Stdout.
 - 如果 target 为目录, 输出到 target/golist.json
 - 如果 target 为 ".json" 文件, 输出到该文件
 - 其它报错

如果未指定参数 `lang` 则取第一个 Godocu 风格的 lang 值.

相关输出结构

```go
// List 表示在同一个 repo 下全部包文档信息
type List struct {
  // Repo 是原源代码所在托管 git 仓库地址.
  // 如果无法识别值为 "localhost"
  Repo string

  // Description 一句话介绍 Repo 或列表
  // Readme 整个 list 的 readme 文件名
  Description, Readme string `json:",omitempty"`

  // 文档文件名
  Filename string
  // Ext 表示除 "go" 格式文档之外的扩展名.
  // 例如: "md text"
  // 该值由使用者手工设置, Godocu 只是保留它.
  Ext string `json:",omitempty"`

  // Subdir 表示文档文件位于 golist.json 所在目录那个子目录.
  // 该值由使用者手工设置, Godocu 只是保留它.
  Subdir string `json:",omitempty"`

  Package []Info // 所有包的信息
}

// Info 表示单个包文档信息.
type Info struct {
  Import   string // 导入路径
  Synopsis string // 一句话包摘要
  // Readme 该包下 readme 文件名
  Readme   string `json:",omitempty"`
  Progress int    // 翻译完成度
}
```

如果 golist.json 已经存在, 那么 Repo, Description, Ext, Subdir 属性被保留.
否则尝试计算 Repo 地址, 如果是官方包, 那么设定 Repo 为 "github.com/golang/go".
如果计算 Repo 失败, 那么设定 Repo 为 "localhost".

*翻译完成度属性 Progress 通过简单比较文档值计算得到,可能与现实不符*

[Example](#example) 段有详细的例子演示如何配套使用.

以 [translations][] 翻译项目为例输出全部包文档清单到 Stdout 的用法有多种:

```shell
$ godocu list -goroot=/path/to/github.com/golang-china/golangdoc.translations ...
$ godocu list /path/to/github.com/golang-china/golangdoc.translations/src...
$ cd /path/to/github.com/golang-china/golangdoc.translations
$ godocu list src...
```

 - 第一种把翻译项目目录当做 `goroot`. "..." 遍历所有包
 - 第二种使用了绝对路径, 注意带上 "/src".
 - 第三种使用了当前路径, 是第二种写法的变种.

输出:

```json
{
    "Repo": "github.com/golang/go",
    "Filename": "doc_zh_CN.go",
    "Package": [
        {
            "Import": "",
            "Synopsis": "tar包实现了tar格式压缩文件的存取.",
            "Progress": 100,
        },
        {
            "Import": "",
            "Synopsis": "zip包提供了zip档案文件的读写服务.",
            "Progress": 95,
        },
        // .....
    ],
}
```


目录关系详见 [Example](#example) 段.

# Replace

指令 `replace` 用 source 的翻译文档替换 target 中未翻译的文档.
要求 source, target 必须都是翻译文档, 即符合 Godocu 文件名命名风格.

显然在使用 `replace` 前, 对 source, target 进行 'merge' 处理可保障代码结构一致.

# Example

这里以第三方包 go-github 为例:

 - 源包 https://github.com/google/go-github/github
 - 翻译 https://github.com/gohub/google

```shell
$ go get github.com/google/go-github/github
```

初次翻译时先生成原源码文档, 假设文档基础路径为 $TARGET, github 上已建立翻译空仓库.

```shell
$ cd $TARGET
$ godocu code -lang=zh_cn github.com/google/go-github/github .
```

此时在 $TARGET 目录下生成:

```
github.com
└── google
    └── go-github
        └── github
            └── doc_zh_CN.go
```

可见源码包和文档的目录树结构是一致的. Godocu 以此计算导入路径.
显然 doc_zh_CN.go 中其实是英文文档. 文档翻译请参见 [golang-china][].

接下来是常规的 git 操作:

```shell
$ cd $TARGET/github.com/google
$ git init
$ git remote add origin git@github.com:gohub/google.git
```

如果在使用 Godocu 之前已经做了翻译, 确保目录结构一致即可. 比如:

```shell
$ cd $TARGET
$ git clone https://github.com/gohub/google ./github.com/google
```

现实中的目录树为:

```
github.com
└── google
    ├── README.md
    ├── go-github
    │   └── github
    │       └── doc_zh_CN.go
    └── golist.json
```


之后就可使用 Godocu 提供的指令进行文档操作了.

实战, 合并 [Go-zh][] 和 [translations][] 的翻译成果.

    merge 中的 source 可能和翻译的版本不符合
    因为 Go-zh 是基于源码的翻译, merge 的 source 不能使用 ./go-zh/src...
    同理 translations 也应选择配套的版本吧
    最佳情况下应该选择相同的 source

下列示意代码假设系统 go 版本和 Go-zh 所用版本一致

```shell
$ cd $TARGET # $TARGET 是此实战工作目录, 先克隆两个项目
$ git clone https://github.com/Go-zh/go go-zh
$ git clone https://github.com/golang-china/golangdoc.translations translations
$ # 类似 builtin 那些需要 -u 参数的包要先单独处理, 目标路径会自动建立
$ godocu code ./go-zh/src/builtin go-zh-trans/src -lang=zh_cn -u
$ # 为 Go-zh 生成文档
$ godocu code ./go-zh/src... go-zh-trans/src -lang=zh_cn
$ # 两个项目都合并最新英文文档, merge 保证了结构一致性
$ godocu merge ... go-zh-trans/src -lang=zh_cn
$ godocu merge ... translations/src -lang=zh_cn
$ # 两种方法进行 replace, 结果可能有所不同
$ godocu replace ./go-zh-trans/src... ./translations/src -lang=zh_cn
$ godocu replace ./translations/src... ./go-zh-trans/src -lang=zh_cn
```

*注意: 同时指定 `target`, `lang` 才会生成(覆盖) target, 不然仅输出在 Stdout.*

两个项目的目录结构可能和最新官方包不一致, 使用 tree 指令对比, 然后手工处理.

[docu]: https://godoc.org/github.com/golang-china/godocu/docu
[golang-china]: https://github.com/golang-china/golang-china.github.com
[Go-zh]: https://github.com/Go-zh/go
[translations]: https://github.com/golang-china/golangdoc.translations