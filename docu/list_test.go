package docu

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

const (
	// OK indicates the lack of an error.

	// testOk 表示没有出错
	testOk = iota
	// testNon non origin
	testNon
)

func TestTransOrigin(t *testing.T) {

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "list_test.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal("Oop!")
	}
	Index(file)

	_decls, _ := declsOf(ConstNum, file.Decls, 0)
	decls := SortDecl(_decls)
	spec, _, _ := decls.SearchSpec("testOk")
	source := testToValueSpec(spec)
	if source == nil {
		t.Fatal("Oop!")
	}
	if source.Doc.Text() != "testOk 表示没有出错\n" {
		t.Fatal(source.Doc.Text())
	}

	origin := transOrigin(file, source.Doc)
	if origin.Text() != "OK indicates the lack of an error.\n" {
		t.Fatal(origin.Text())
	}
	spec, _, _ = decls.SearchSpec("testNon")
	target := testToValueSpec(spec)
	if target == nil {
		t.Fatal("Oop!")
	}
	origin = transOrigin(file, target.Doc)
	if origin != nil {
		t.Fatal(origin.Text())
	}
	want := source.Doc.Text() +
		GoDocu_Dividing_line + "\n" +
		target.Doc.Text()

	replaceDoc(file, file, target.Doc, source.Doc)
	got := target.Doc.Text()
	if got != want {
		t.Fatal(got)
	}
}

func testToValueSpec(spec ast.Spec) (vs *ast.ValueSpec) {
	if spec != nil {
		vs, _ = spec.(*ast.ValueSpec)
	}
	return
}
