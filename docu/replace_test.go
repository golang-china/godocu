package docu

import (
	"bytes"
	"testing"
)

func TestReplace(t *testing.T) {
	var buf bytes.Buffer
	source := testParseFile(t, "testdata/replace_source.text")
	target := testParseFile(t, "testdata/merge_origin_trans.text")
	source.Unresolved = godocuStyle
	target.Unresolved = godocuStyle

	Replace(target, source)

	err := Fprint(&buf, target)
	if err != nil {
		t.Fatal(err)
	}

	text := buf.String()
	_, err = testWantDiffer(t, "testdata/replace_source_merge_origin_trans.text").WriteString(text)
	if err != nil {
		t.Fatalf("%s\n%s\n%s", err, "TEXT:", text)
	}
}
