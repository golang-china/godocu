# GoDocu

Godocu 基于 [docu] 实现的指令行工具, 从 Go 源码提取并生成文档.

功能:

  - 80 列换行, 支持多字节字符
  - 若原注释已经符合 80 列换行, 保持不变.
  - 内置两种文档风格, Go 源码风格和 godoc 文本风格
  - 可提取执行包文档, 测试包文档, 非导出符号文档
  - 遍历目录
  - 生成文档概要清单
  - 合并不同版本文档
  - 简单比较包文档的不同之处

该工具在 Golang 官方包下测试通过, 非官方包请核对输出结果.

命名风格:

Godocu 生成的文档文件名由包名称和 `lang` 参数计算得出, 格式为 `prefix_lang.ext`.

前缀由包名称计算得到 `doc`, `main` 或 `test`.

如果参数 `lang` 非空, 添加后缀 `_lang`.

扩展名

 - `code`,`merge` 指令输出扩展名为 ".go".
 - `plain` 指令扩展名为 ".text".

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
    list    prints godocu style documents list
    merge   merge source doc to target

The source are:

    package import path or absolute path
    the path to a Go source file

The target are:

    the directory as an absolute base path for compare or prints

The arguments are:

  -cmd
      show symbols with package docs even if package is a command
  -gopath string
      specifies gopath (default $GOPATH)
  -goroot string
      Go root directory (default $GOROOT)
  -lang string
      the lang pattern for the output file, form like en or zh_CN
  -test
      show symbols with package docs even if package is a testing
  -u  show unexported symbols as well as exported
```

# source

source 用于计算 go 源码文件路径, 可以是 import path 或绝对路径表示的目录或文件.
如果是 import path, Godocu 会在 `GOROOT/src`, `GOPATH/src` 下查找并计算出绝对路径.

非文件 source 可以后缀 `...` 表示遍历子目录.
若 source 为 `...` 表示所有官方包, 即遍历 `GOROOT/src` 下的所有包.

多数指令中 source 用来计算 import path, 这要求计算后的绝对路径要包含 "/src/".

详情参见相关指令以及 `Example` 段.

# target

target 在 `diff`,`first`,`tree`,`code`,`plain`,`merge` 指令中表示绝对的基本路径.
拼接 source 中计算出的 import path 后得到目标绝对路径.

这意味着某个包的目录结构在 source 和 target 中是相同的.

对于 `diff`, `first`, `tree` 指令, target 必选, 表示对比目标, 输出到 Stdout.

对于 `merge' 指令, target 必选, 表示目标文档.

对于 `code`, `plain` 指令, target 可选, 表示结果目标, 缺省输出到 Stdout.

对于 `list` 指令 target 有独立含义.

详情参见相关指令以及 `Example` 段.

# lang

参数 `lang` 指定输出文件名后缀, 格式为 lang 或 lang_ISOCountryCode.
即 lang 部分为小写, ISOCountryCode 部分为大写.

辅助函数 `docu.LangNormal` 提供规范化处理.

某些情况下即便未指定参数 `lang`, godouc 会通过 target 中现存的文件名计算得到.

详情参见相关指令.

# cmd

参数 'cmd' 允许操作 `main` 包顶级导出声明.

# unexported

参数 'u' 允许操作顶级非导出声明, 现实中有这样的需求. 比如 `builtin` 包的声明都是非导出的, 但其文档在 Go 文档中是不可或缺的.

也许某个文档仅需要包含特别的非导出声明, Godocu 的非导出优先策略是:

 1. 已存在符合 docu 命名风格的 ".go" 文档, 目标中的非导出声明被保留
 2. 否则按是否使用了 'u' 参数处理.

# Code

指令 `code`, `plain` 输出格式化文档.

方便起见, 当 target 值为 "--", 表示输出到原包目录.

如果指定了 target 要求参数 `lang` 非空.

# Tree

指令 `tree` 遍历比较并输出 sourec, target 目录结构的不同.

遍历目录时 Godocu 参照 Go 命名习惯, 忽略 `testdata`, `vendor` 之类的目录.

遍历比较当前版本 1.6.2 和老版本的差异

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

指令 `diff` 比较 source, target 共有的包并输出差异信息, 而 `first` 仅输出首个差异信息.

要求由 source 计算出的绝对路径必须包含 "/src/".

比较 reflect 在当前版本 1.6.2 和老版本的差异

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

比较 go/types 在当前版本 1.6.2 和老版本的差异

```shell
$ godocu diff go/types /usr/local/Cellar/go/1.5.3/libexec/src
```

输出

```
TEXT:
    Package types declares the data types and implements
    the algorithms for type-checking of Go packages. Use
    Config.Check to invoke the type checker for a package.
    Alternatively, create a new type checked with NewChecker
    and invoke it incrementally by calling Checker.Files.

    Type-checking consists of several interdependent phases:

    Name resolution maps each identifier (ast.Ident) in the program to the
    language object (Object) it denotes.
    Use Info.{Defs,Uses,Implicits} for the results of name resolution.

    Constant folding computes the exact constant value (constant.Value)
    for every expression (ast.Expr) that is a compile-time constant.
    Use Info.Types[expr].Value for the results of constant folding.

    Type inference computes the type (Type) of every expression (ast.Expr)
    and checks for compliance with the language specification.
    Use Info.Types[expr].Type for the results of type inference.

    For a tutorial, see https://golang.org/s/types-tutorial.
DIFF:
    Package types declares the data types and implements
    the algorithms for type-checking of Go packages. Use
    Config.Check to invoke the type checker for a package.
    Alternatively, create a new type checked with NewChecker
    and invoke it incrementally by calling Checker.Files.

    Type-checking consists of several interdependent phases:

    Name resolution maps each identifier (ast.Ident) in the program to the
    language object (Object) it denotes.
    Use Info.{Defs,Uses,Implicits} for the results of name resolution.

    Constant folding computes the exact constant value (constant.Value)
    for every expression (ast.Expr) that is a compile-time constant.
    Use Info.Types[expr].Value for the results of constant folding.

    Type inference computes the type (Type) of every expression (ast.Expr)
    and checks for compliance with the language specification.
    Use Info.Types[expr].Type for the results of type inference.

TEXT:
    import (
        "bytes"
        "container/heap"
        "fmt"
        "go/ast"
        "go/constant"
        "go/parser"
        "go/token"
        "io"
        "math"
        "sort"
        "strconv"
        "strings"
        "sync"
        "testing"
        "unicode"
    )
DIFF:
    import (
        "bytes"
        "container/heap"
        "fmt"
        "go/ast"
        "go/constant"
        "go/parser"
        "go/token"
        "io"
        "math"
        "path"
        "sort"
        "strconv"
        "strings"
        "testing"
        "unicode"
    )

TEXT:
    func (*Config) Check(path string, fset *token.FileSet, files []*ast.File, info *Info)
    (*Package, error)

    Check type-checks a package and returns the resulting package object and
    the first error if any. Additionally, if info != nil, Check populates each
    of the non-nil maps in the Info struct.

    The package is marked as complete if no errors occurred, otherwise it is
    incomplete. See Config.Error for controlling behavior in the presence of
    errors.

    The package is specified by a list of *ast.Files and corresponding
    file set, and the package path the package is identified with.
    The clean path must not be empty or dot (".").
DIFF:
    func (*Config) Check(path string, fset *token.FileSet, files []*ast.File, info *Info)
    (*Package, error)

    Check type-checks a package and returns the resulting package object,
    the first error if any, and if info != nil, additional type information.
    The package is marked as complete if no errors occurred, otherwise it is
    incomplete. See Config.Error for controlling behavior in the presence of
    errors.

    The package is specified by a list of *ast.Files and corresponding
    file set, and the package path the package is identified with.
    The clean path must not be empty or dot (".").

TEXT:
    func (*Package) SetName(name string)
DIFF:
    none

FROM: package go/types
```

go 1.6.2 的 Doc 注释多了一行

    For a tutorial, see https://golang.org/s/types-tutorial.

和其它一些变化.

如果看到的不是 `TEXT:` 而是 `FORM:` 表示折叠为一行后值相同, 即格式发生变化,

# List

list 指令以 JSON 格式输出 Godocu 风格文档清单.

source 中 Godocu 风格文档才会出现在清单中.

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
  Repo        string // 托管 git 仓库地址.
  Description string // 一句话介绍 Repo 或列表
  Subdir      string // 文档所在 repo 下的子目录
  Lang        string // 同一个列表具有相同的 Lang
  Markdown    bool   // 是否具有完整的 Markdown 文档
  Info        []Info
}

// Info 表示单个包文档信息.
type Info struct {
  Import   string // 导入路径
  Synopsis string // 一句话包摘要
  Progress int    // 翻译完成度
  Prefix   string // 例如 "doc" 或 "doc,main,test"
}
```

通常纯粹的文档不应该位于 `GOPATH` 之下, 而 list 需要正确计算出每个包的导入路径,
即 Info.Import 属性. list  按以下优先级进行导入路径处理:

 - 如果 golist.json 已经存在, 以其中的 Repo, Subdir 值与本地绝对路径进行计算
 - 如果本地绝对路径含 "src" 目录, "src" 后面的就是导入路径
 - 在绝对路径中搜索常见的仓库托管服务域名计算导入路径
 - 生成空值的 golist.json 提示使用者手工设置 Repo, Subdir

上例中 golang-china 的翻译项目包含 'src' 子目录, Godocu 可以凭此计算出导入路径.
对于不含有 'src' 的翻译, Godocu 有可能计算错误, 可以通过预先建立 `golist.json`,
并设置 `Repo`,`Description`,`Subdir` 属性, Godocu 凭此计算其它参数.

Example 段有详细的例子演示如何配套使用.

以 golang-china 的翻译项目为例输出全部包文档清单到  Stdout 有三种用法:

```shell
$ godocu list -goroot=/path/to/github.com/golang-china/golangdoc.translations ...
$ godocu list /path/to/github.com/golang-china/golangdoc.translations/src...
$ cd /path/to/github.com/golang-china/golangdoc.translations/src
$ godocu list ....
```

 - 第一种把翻译项目目录当做 `goroot`.
 - 第二种使用了绝对路径, 注意带上 "/src".
 - 第三种使用了当前路径, 是第二种写法的变种.

输出:

```json
{
    "Repo": "",
    "Description": "",
    "Subdir": "",
    "Lang": "",
    "Markdown": true,
    "Info": [
        {
            "Import": "",
            "Synopsis": "tar包实现了tar格式压缩文件的存取.",
            "Progress": 100,
            "Prefix": "doc"
        },
        {
            "Import": "",
            "Synopsis": "zip包提供了zip档案文件的读写服务.",
            "Progress": 95,
            "Prefix": "doc"
        },
        // .....
    ],
}
```


目录关系详见 `Example` 段.

# Merge

merge 指令对两个相同导入路径的包文档进行合并. 细节:

 - source 可以是源码或包文档.
 - target 必须是包文档, 源码包被忽略.
 - 依照 target 中的声明过滤 source, 参数 `cmd`,`test`,'u' 失去作用.
 - 如果 target 没有 import, 添加 source 的 import.
 - 匹配 source, target 中相同的顶级声明, 合并 source 的文档在 target 前面.
 - 指定 `lang` 参数才生成或覆盖 target, 否则仅向 stdout 打印结果.

合并 `builtin` 包文档到 golang-china 的翻译项目.

```shell
$ godocu merge builtin /path/to/github.com/golang-china/golangdoc.translations/src
```

遍历所有官方包文档合并到 golang-china 的翻译项目.

```shell
$ godocu merge ... /path/to/github.com/golang-china/golangdoc.translations/src
```

例子中的 target 含有子目录 "src", 并以它结尾, 这不是必须的.


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

显然 Godocu 生成的目录结构是带完整导入路径的, 那么接下来的 git 操作为:

```shell
$ cd $TARGET/github.com/google
$ git init
$ git remote add origin git@github.com:gohub/google.git
```

如果在使用 Godocu 之前已经做了翻译, 保持目录结构与完整导入路径对应即可. 比如:

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

显然子目录树结构在原源码包和翻译文档中必须保持一致.

[docu]: https://godoc.org/github.com/golang-china/godocu/docu