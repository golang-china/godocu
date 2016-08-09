# GoDocu

godocu 基于 [docu] 实现的指令行工具, 从 Go 源码提取并生成文档.

功能:

  - 80 列换行, 支持多字节字符
  - 内置两种文档风格, Go 源码风格和 godoc 文本风格
  - 可提取执行包文档, 测试包文档, 非导出符号文档
  - 文档概要清单
  - 简单比较包文档的不同之处
  - 遍历目录
  - 合并不同版本文档
  - 若原文档已经符合 80 列换行, 保持不变.

该工具在 Golang 官方包下测试通过, 非官方包请核对输出结果.

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
      specifies gopath (default "/Users/achun/Workspace/gowork")
  -goroot string
      Go root directory (default "/usr/local/Cellar/go/1.6/libexec")
  -lang string
      the lang pattern for the output file, form like en or zh_CN
  -test
      show symbols with package docs even if package is a testing
  -u  show unexported symbols as well as exported
```

# source

source 用于计算 go 源码文件, 可以是绝对路径表示的目录或者文件.
如果是 import path, godocu 会在 GOROOT, GOPATH 中查找并计算出绝对路径.

非独立文件 source 可以后缀 `...` 表示遍历子目录.
独立的 `...` 表示所有官方包, 即计算后的 `GOROOT/src` 下所有包.

# target

对于 `diff`, `first`, `list` 指令, target 是文件或目录, 输出到 Stdout.

对于 `code`, `plain` , `merge' 指令, target 是生成文档基础路径,
子路径和文件名由 source 和 `lang` 参数计算得出.

文件名前缀由包名称计算得到 `doc`, `main` 或 `test`.

如果参数 `lang` 非空, 添加后缀 `_{lang}`. 这是 docu 的命名风格.

扩展名

 - `code`,`merge` 指令输出扩展名为 ".go".
 - `plain` 指令扩展名为 ".text".

# lang

参数 `lang` 指定输出文件名后缀, 格式为 lang 或 lang_ISOCountryCode.
即 lang 部分为小写, ISOCountryCode 部分为大写.

辅助函数 `docu.LangNormal` 提供规范化处理.

由于文件名有固定格式, godouc 会通过 target 中已存在的文件名计算得到.

如果 `lang` 非空, 新建或覆盖计算后的目标文件.
如果 `lang` 为空, 且目标文件符合 docu 命名风格, 目标文件被覆盖.

# unexported

参数 'u' 允许输出非导出顶级声明, 现实中有这样的需求. 比如 `builtin` 包的声明都是非导出的, 但其文档在 Go 文档中是不可或缺的.

也许某个文档仅需要包含特别的非导出声明, Godocu 的非导出优先策略是:

 1. 已存在符合 docu 命名风格的 ".go" 文档, 目标中的非导出声明被保留
 2. 否则按是否使用了 'u' 参数处理.

# Code

指令 `code`, `plain` 输出格式化文档.

方便起见, 当 target 值为 "--", 表示同目录输出, 这有可能是覆盖.

# Diff

指令 `first` 比较两个包, 输出首个差异信息, 而 `diff` 输出全部差异信息.

要求由 source,target 计算出的绝对路径必须包含 "/src/".

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

遍历比较 "cmd" 及其子目录

```shell
$ godocu diff cmd... /usr/local/Cellar/go/1.5.2/libexec/src/
```

因目录结构不同, 不进行文档对比.

输出:

```
source: /usr/local/Cellar/go/1.6.2/libexec/src/cmd
target: /usr/local/Cellar/go/1.5.2/libexec/src/

source target import_path
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
  none  path  cmd/internal/rsc.io
  none  path  cmd/internal/rsc.io/arm
  none  path  cmd/internal/rsc.io/arm/armasm
  none  path  cmd/internal/rsc.io/x86
  none  path  cmd/internal/rsc.io/x86/x86asm
  none  path  cmd/vet/whitelist
```

# List

list 指令以 JSON 格式输出 Godocu 风格文档清单.

source 中 Godocu 风格文档才会出现在清单中.

target:

 - 如果 target 为空, 输出到 Stdout.
 - 如果 target 为目录, 输出到 target/golist.json
 - 如果 target 为 ".json" 文件, 输出到该文件
 - 其它报错

如果未指定参数 `lang` 则取第一个 Godocu 风格的 lang 值.

通常纯粹的文档不应该位于 `GOPATH` 之下, 需要适当的使用 `goroot`,`gopath` 参数.

以 golang-china 的翻译项目为例输出全部包文档清单有三种写法:

```shell
$ godocu list -goroot=/path/to/github.com/golang-china/golangdoc.translations ...
$ godocu list /path/to/github.com/golang-china/golangdoc.translations/src...
$ cd /path/to/github.com/golang-china/golangdoc.translations/src
$ godocu list ....
```

 - 第一种写法是把翻译项目目录当做 `goroot`.
 - 第二种写法则使用绝对路径.
 - 第三种写法使用了相对路径, 其实是第二种写法的变种.

上例中 golang-china 的翻译项目包含 'src' 子目录, Godocu 可以凭此计算出导入路径.
对于不含有 'src' 的翻译, Godocu 有可能计算错误, 可以通过预先建立 `golist.json`,
并设置 `Repo`,`Description`,`Subdir` 属性, 且计算后的本地绝对中必须含有 `/Repo/Subdir/`,  Godocu 凭此计算导入路径.

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

显然 godocu 生成的目录结构是带完整导入路径的, 那么接下来的 git 操作为:

```shell
$ cd $TARGET/github.com/google
$ git init
$ git remote add origin git@github.com:gohub/google.git
```

如果在使用 godocu 之前已经做了翻译, 保持目录结构与完整导入路径对应即可. 比如:

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