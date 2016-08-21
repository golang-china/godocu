package docu

import "testing"

func TestIsNormalName(t *testing.T) {
	tests := []struct {
		want bool
		name string
	}{
		{false, "doc"},
		{true, "doc.go"},
		{true, "doc.md"},
		{true, "main_zh_CN.md"},
		{true, "test_zh.go"},
		{false, "test_zh_cn.go"},
		{false, "test_CN.go"},
	}
	for _, tt := range tests {
		if got := IsNormalName(tt.name); got != tt.want {
			t.Errorf("%q. IsNormalName() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s   string
		sep string
	}{
		{goarchList, "s390x"},
		{goosList, "android"},
		{goosList, "windows"},
	}
	for _, tt := range tests {
		if !contains(tt.s, tt.sep) {
			t.Errorf("contains(%q,%q)", tt.s, tt.sep)
		}
	}
}

func TestIsOSArchFile(t *testing.T) {
	tests := []string{
		"zsyscall_linux_s390x.go",
		"doc_linux.go",
	}
	for _, name := range tests {
		if !IsOSArchFile(name) {
			t.Errorf("IsOSArchFile(%q)", name)
		}
	}
}
