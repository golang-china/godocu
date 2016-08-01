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
