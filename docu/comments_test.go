package docu

import (
	"go/ast"
	"testing"
)

const (
	// OK indicates the lack of an error.

	// testOk 表示没有出错
	testOk = iota
	// testNon non origin
	testNon
)

type (
	testComment struct {
		// origin

		// trans
		Comment string
	}
)

func TestOriginDoc(t *testing.T) {
	file := testParseFile(t, "comments_test.go")
	_decls, _ := declsOf(TypeNum, file.Decls, 0)
	decls := SortDecl(_decls)

	spec, _, _ := decls.SearchSpec("testComment")
	if spec == nil {
		t.Fatal("Oop! type testComment is nil")
	}
	ts, _ := spec.(*ast.TypeSpec)
	if ts == nil {
		t.Fatal("Oop! type testComment is nil")
	}
	st, _ := ts.Type.(*ast.StructType)
	if st == nil {
		t.Fatal("Oop! type testComment is nil")
	}
	field, _ := findField(st.Fields, "Comment")
	if field == nil {
		t.Fatal("Oop! testComment.Comment is nil")
	}
	origin := OriginDoc(file.Comments, field.Doc)
	if origin.Text() != "origin\n" || field.Doc.Text() != "trans\n" {
		t.Fatal(origin.Text(), field.Doc.Text())
	}

	_decls, _ = declsOf(ConstNum, file.Decls, 0)
	decls = SortDecl(_decls)

	spec, _, _ = decls.SearchSpec("testOk")
	sdoc := SpecDoc(spec)
	if sdoc == nil {
		t.Fatal("Oop!")
	}
	if sdoc.Text() != "testOk 表示没有出错\n" {
		t.Fatal(sdoc.Text())
	}
	origin = OriginDoc(file.Comments, sdoc)
	if origin.Text() != "OK indicates the lack of an error.\n" {
		t.Fatal(origin.Text())
	}

	spec, _, _ = decls.SearchSpec("testNon")
	tdoc := SpecDoc(spec)
	if tdoc == nil {
		t.Fatal("Oop!")
	}
	origin = OriginDoc(file.Comments, tdoc)
	if origin != nil {
		t.Fatal(origin.Text())
	}
	want := tdoc.Text() +
		GoDocu_Dividing_line + "\n" +
		sdoc.Text()

	replaceDoc(file, file, tdoc, sdoc)
	got := tdoc.Text()
	if got != want {
		t.Fatalf("WANT:\n%s\nDIFF:\n%s", want, got)
	}
}
