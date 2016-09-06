package docu

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"testing"
)

type (
	testTypeOrigin struct {
		// a
		a []string // a
		// b
		b string // b
		// cd
		c, d string // cd
		e    string
		testing.T
	}

	testTypeTrans struct {
		// trans a
		a []string // trans a
		// trans b
		b string // b // trans b
		// trans cd
		c    string // trans cd
		d, e string
	}
)

const testmergeTypeWant = `type (
	testTypeOrigin struct {
		// a

		// trans a
		a []string // a

		// b

		// trans b
		b string // b // trans b

		// cd

		// trans cd
		c, d string // cd
		e    string
		testing.T
	}

	testTypeTrans struct {
		// trans a
		a []string // trans a

		// trans b
		b string // b // trans b

		// trans cd
		c    string // trans cd
		d, e string
	}
)`

func specStructType(spec ast.Spec) *ast.StructType {
	if spec == nil {
		return nil
	}
	ts, _ := spec.(*ast.TypeSpec)
	if ts == nil {
		return nil
	}
	st, _ := ts.Type.(*ast.StructType)
	return st
}

func TestMergeFieldsDoc(t *testing.T) {
	file := testParseFile(t, "merge_test.go")
	_decls, _ := declsOf(TypeNum, file.Decls, 0)
	decls := SortDecl(_decls)
	spec, decl, _ := decls.SearchSpec("testTypeOrigin")
	st := specStructType(spec)
	if st == nil {
		t.Fatal("Oop!")
	}

	tests := []struct {
		name    string
		doc     string
		comment string
		pos     int
	}{
		{"a", "a\n", "a\n", 0},
		{"b", "b\n", "b\n", 0},
		{"c", "cd\n", "cd\n", 0},
		{"d", "cd\n", "cd\n", 1},
		{"e", "", "", 0},
	}
	for _, tt := range tests {
		field, i := findField(st.Fields, tt.name)
		if field == nil || i != tt.pos {
			t.Fatalf("findField(%q) %d", tt.name, i)
		}
		text := field.Doc.Text()

		if text != tt.doc {
			t.Fatalf("findField(%q) %d doc %q", tt.name, i, text)
		}

		text = field.Comment.Text()
		if text != tt.comment {
			t.Fatalf("findField(%q) %d comment %q", tt.name, i, text)
		}
	}

	tspec, _, _ := decls.SearchSpec("testTypeTrans")
	tt := specStructType(tspec)
	if tt == nil {
		t.Fatal("Oop!")
	}
	// 注意 target, source 次序
	mergeFieldsDoc(tt.Fields, st.Fields)
	tests = []struct {
		name    string
		doc     string
		comment string
		pos     int
	}{
		{"a", "a\n___GoDocu_Dividing_line___\ntrans a\n", "a\n", 0},
		{"b", "b\n___GoDocu_Dividing_line___\ntrans b\n", "b // trans b\n", 0},
		{"c", "cd\n___GoDocu_Dividing_line___\ntrans cd\n", "cd\n", 0},
		{"d", "cd\n___GoDocu_Dividing_line___\ntrans cd\n", "cd\n", 1},
		{"e", "", "", 0},
	}

	for _, tt := range tests {
		field, i := findField(st.Fields, tt.name)
		if field == nil || i != tt.pos {
			t.Fatalf("findField(%q) %d", tt.name, i)
		}
		text := field.Doc.Text()

		if text != tt.doc {
			t.Fatalf("findField(%q) %d doc %q", tt.name, i, text)
		}

		text = field.Comment.Text()
		if text != tt.comment {
			t.Fatalf("findField(%q) %d comment %q", tt.name, i, text)
		}
	}

	var buf bytes.Buffer
	err := FprintGenDecl(&buf, decl, nil)
	if err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	_, err = NewDiffer("Differ", testmergeTypeWant).WriteString(text)
	if err != nil {
		t.Fatalf("%s\n%s\n%s", err, "TEXT:", text)
	}
}

func TestMergeDeclsDoc(t *testing.T) {
	var buf bytes.Buffer
	origin := testParseFile(t, "testdata/origin.go")
	ExportedFileFilter(origin)
	Fprint(&buf, origin)
	text := buf.String()
	_, err := testWantDiffer(t, "testdata/code_origin.text").WriteString(text)
	if err != nil {
		t.Fatalf("%s\n%s\n%s", err, "TEXT:", text)
	}

	trans := testParseFile(t, "testdata/trans.go")
	MergeDoc(trans.Doc, origin.Doc)
	MergeDeclsDoc(trans.Decls, origin.Decls)
	buf.Reset()
	Fprint(&buf, origin)
	text = buf.String()
	_, err = testWantDiffer(t, "testdata/merge_origin_trans.text").WriteString(text)
	if err != nil {
		t.Fatalf("%s\n%s\n%s", err, "TEXT:", text)
	}
}

func testParseFile(t *testing.T, name string) *ast.File {
	file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ParseComments)
	if err != nil {
		t.Fatal("Oop! ParseFile", name, err.Error())
	}
	Index(file)
	return file
}

func testWantDiffer(t *testing.T, name string) *Differ {
	bs, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(name, err.Error())
	}
	return NewDiffer(name, string(bs))
}
