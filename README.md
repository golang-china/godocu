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
  -test
      show symbols with package docs even if package is a testing
  -u  show unexported symbols as well as exported
```

# Example

比较 fmt 在两个版本中的不同

```shell
$ godocu -goroot=/usr/local/Cellar/go/1.6/libexec -diff fmt /usr/local/Cellar/go/1.5.2/libexec/src
[TEXT] package doc, on package fmt
```

意思是

```
[内容发生变化] package 文档不同, 在 fmt 包
```

```shell
$ godocu -goroot=/usr/local/Cellar/go/1.6/libexec -diff reflect /usr/local/Cellar/go/1.5.2/libexec/src
[TEXT] Decls,at package reflect
```

意思是

```
[内容发生变化] 顶级声明不同, 在 reflect 包
```

[docu]: https://godoc.org/github.com/golang-china/godocu/docu