package docu

import (
	"path/filepath"
	"testing"
)

func TestAbs(t *testing.T) {
	name := `github.com/golang-china/godocu/docu`
	abs := Abs(name)
	if abs != filepath.Join(GOPATHS[0], "src", name) {
		t.Fatal(abs)
	}
	abs = Abs(".")
	if abs != filepath.Join(GOPATHS[0], "src", name) {
		t.Fatal(abs)
	}
	name = "go"
	abs = Abs(name)
	if abs != filepath.Join(GOROOT, "src", name) {
		t.Fatal(abs)
	}
}
