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

该工具在 Golang 官方包下测试通过, 非官方包请核对输出结果.

# Install

```
go get github.com/golang-china/godocu
```

# Usage

```
usage: godocu package [target]
         target       the directory as an absolute base path of docs.
                      the path for output if not set -diff.
  -cmd
      show symbols with package docs even if package is a command
  -diff
      list different of package of target-path docs
  -go
      prints a formatted string to standard output as Go source code
  -gopath string
      specifies gopath (default $GOPATH)
  -goroot string
      Go root directory (default $GOROOT)
  -lang string
      the lang pattern for the output file (default "origin")
  -test
      show symbols with package docs even if package is a testing
  -u  show unexported symbols as well as exported
```

# Diff

可选参数 `-diff` 可比较两个包, 输出首个差异信息.

比较 reflect 在两个版本中的不同

```shell
$ godocu -goroot=/usr/local/Cellar/go/1.5.2/libexec -diff reflect /usr/local/Cellar/go/1.6.2/libexec/src
```

输出

```
TEXT:
    Decls length 112
DIFF:
    Decls length 113
FROM: package reflect

////////////////////////////////////////////////////////////////////////////////
```

意思是

```
[内容]:
    顶级声明长度 112
[不同]:
    顶级声明长度 113
来自: package reflect
```


```shell
$ godocu -goroot=/usr/local/Cellar/go/1.5.2/libexec -diff go/types /usr/local/Cellar/go/1.6.2/libexec/src
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

    For a tutorial, see https://golang.org/s/types-tutorial.

FROM: package go/types
```


go 1.6.2 的注释多了一行 `For a tutorial, see https://golang.org/s/types-tutorial.`.


如果看到的不是 `TEXT:` 而是 `FORM:` 表示折叠为一行后值相同, 即格式发生变化,

遍历

```shell
$ godocu go...
```

遍历比较

```shell
$ godocu -goroot=/usr/local/Cellar/go/1.5.2/libexec -diff go... /usr/local/Cellar/go/1.6/libexec/src/
```

如果目录结构不同, 只输出结构不同的, 不进行文档对比.

[docu]: https://godoc.org/github.com/golang-china/godocu/docu