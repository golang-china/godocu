package docu

const MarkdownTemplate = `{{define "echo"}}
` + "```go" + `
{{.}}
` + "```" + `
{{end}}{{/*
此模板输出 Markdown 格式.
模板传入 Data 实例作为模板执行数据. 并映射了 docu.FuncsMap.
*/}}{{if eq .Key .ImportPath}}{{/*
模板必须通过 Type 方法(任意位置)设定输出文件扩展名, 否则会抛弃输出.
在这个例子中只输出标准的 doc 文档, 忽略 main, test 文档.
*/}}{{$.Type "md"}}{{$this := .File}}# {{base .ImportPath}}

{{/*
函数 progress 返回文档翻译完成度, 值为 0-100. 该值有多种用途.
如果非 0 显示完成度, 并不输出原语言文档. 如果为 0 等同没有翻译, 不显示.
*/}}{{if $trans := progress $this}}Translation Progress: {{$trans}}

{{end}}{{/*
函数 canonicalImportPaths 返回文档权威导入路径.
*/}}{{if $x := canonicalImportPaths $this}}{{template "echo" $x}}{{end}}{{/*
主文档以及各种声明
*/}}{{if $this.Doc}}{{wrap $this.Doc.Text}}{{end}}{{/*

常量
*/}}{{range $i, $x := decls $this.Decls .CONST}}{{if eq $i 0}}
## const

{{end}}{{$.Text $x}}{{template "echo" $.Code $x}}{{end}}{{/*
*/}}{{range $i, $x := decls $this.Decls .VAR}}{{if eq $i 0}}
## var

{{end}}{{$.Text $x}}{{template "echo" $.Code $x}}{{end}}{{/*

由于未实现常规排序, 只能采取分步剔除的方法
*/}}{{$fs:=decls $this.Decls .FUNC}}{{range $i, $x := decls $this.Decls .TYPE}}{{if eq $i 0}}
## type

{{end}}{{$lit := identLit $x}}
### {{$lit}}

{{$.Text $x}}{{template "echo" $.Code $x}}{{/*
构造函数
*/}}{{$pos := indexConstructor $fs $lit}}{{if ne -1 $pos}}{{$x := index $fs $pos}}{{/*

*/}}{{clear $fs $pos}}{{$.Text $x}}{{template "echo" $.Code $x}}{{end}}{{/*
成员方法
*/}}{{range $m := methods $this.Decls $lit}}
### {{identLit $m | starLess}}

{{$.Text $m}}{{template "echo" $.Code $m}}{{end}}{{end}}{{/*

函数
*/}}{{range $i, $x := trimRight $fs}}{{if eq $i 0}}
## func

{{end}}{{if $x}}
### {{identLit $x}}

{{$.Text $x}}{{template "echo" $.Code $x}}{{end}}{{end}}{{/*
*/}}{{if $x := license $this}}
# License

{{wrap $x}}
{{end}}{{end}}`
