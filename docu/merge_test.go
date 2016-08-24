package docu

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

type (
	testTypeSource struct {
		// a
		a []string // a
		// b
		b string // b
		// cd
		c, d string // cd
		e    string
	}

	testTypeTrans struct {
		// trans a
		a []string // trans a
		// trans b
		b string // trans b
		// trans cd
		c    string // trans cd
		d, e string
	}
)

const testmergeTypeWant = `struct {
    // trans a

    // a
    a   []string // a
    // trans b

    // b
    b   string // b
    // trans cd

    // cd
    c    string // cd
    d, e string
}`

func TestMergeFieldsDoc(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "merge_test.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatal("Oop!")
	}
	Index(file)
	_decls, _ := declsOf(TypeNum, file.Decls, 0)
	decls := SortDecl(_decls)
	spec, _, _ := decls.SearchSpec("testTypeSource")
	if spec == nil {
		t.Fatal("Oop!")
	}

	st, _ := spec.(*ast.TypeSpec).Type.(*ast.StructType)
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
	if tspec == nil {
		t.Fatal("Oop!")
	}

	tt, _ := tspec.(*ast.TypeSpec).Type.(*ast.StructType)
	if tt == nil {
		t.Fatal("Oop!")
	}
	mergeFieldsDoc(st.Fields, tt.Fields)
	var buf bytes.Buffer
	config.Fprint(&buf, fset, tt)
	text := strings.Replace(buf.String(), "    //"+GoDocu_Dividing_line, "", -1)
	if text != testmergeTypeWant {
		t.Fatal(text)
	}
}
