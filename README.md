# GoDocu

godocu 基于 [docu] 实现的命令行工具, 从 Go 源码提取并生成文档.

功能:

  - 多字节文档超长换行
  - 生成 godoc 文本风格文档
  - 生成 Go 源码风格文档
  - 可提取执行包文档
  - 可提取非导出符号文档
  - 可提取测试包文档

# Install

```
go get github.com/golang-china/godocu
```

# Usage

```
usage: godocu package
  -cmd
        show symbols with package docs even if package is a command
  -go
        prints a formatted string to standard output as Go source code
  -gopath string
        specifies gopath (default $GOPATH)
  -goroot string
        Go root directory (default $GOROOT)
  -test
        show symbols with package docs even if package is a testing
  -u
        show unexported symbols as well as exported
```


[docu]: https://godoc.org/github.com/golang-china/godocu/docu