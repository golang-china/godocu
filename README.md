# GoDocu

godocu 基于 [docu] 实现的命令行工具, 从 Go 源码提取并生成文档.

功能:

  - 多字节文档超长换行
  - 生成 godoc 文本风格文档
  - 生成 Go 源码风格文档
  - 可提取执行包文档
  - 可提取非导出符号文档
  - 可提取测试包文档
  - 简单比较包文档的不同之处
  - 遍历目录
  - 合并不同版本的注释

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
      the lang pattern for the output file, form like xx[_XX] (default "origin")
  -test
      show symbols with package docs even if package is a testing
  -u  show unexported symbols as well as exported
```

# Diff

命令 `first` 比较两个包, 输出首个差异信息, `diff` 输出全部差异信息.

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
    Type ImportMode
DIFF:
    none
TEXT:
    Type ImporterFrom
DIFF:
    none
TEXT:
    Type Struct struct{fields []*Var; tags []string; offsets []int64; offsetsOnce sync.Once}
DIFF:
    Type Struct struct{fields []*Var; tags []string; offsets []int64}
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


go 1.6.2 的 Doc 注释多了一行 `For a tutorial, see https://golang.org/s/types-tutorial.`.
其他还有一些变化.

如果看到的不是 `TEXT:` 而是 `FORM:` 表示折叠为一行后值相同, 即格式发生变化,

遍历

```shell
$ godocu code go...
```

遍历比较 "cmd" 以及子目录

```shell
$ godocu diff cmd... /usr/local/Cellar/go/1.5.2/libexec/src/
```

输出:

因目录结构不同, 只输出不同的目录, 不进行文档对比.

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

# Merge

merge 命令对两个相同导入路径的包文档进行合并. 细节:

 - 只有 source, target 中相同的顶级声明文档会被合并.
 - source 的文档在 target 顶部
 - 如果 target 没有 import, 添加 source 的 import


[docu]: https://godoc.org/github.com/golang-china/godocu/docu