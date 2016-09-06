package docu

import "testing"

func TestIsNormalName(t *testing.T) {
	tests := []struct {
		want bool
		name string
	}{
		{false, "doc"},
		{false, "doc.go"},
		{false, "doc.md"},
		{true, "main_zh_CN.md"},
		{true, "test_zh.go"},
		{false, "test_zh_cn.go"},
		{false, "test_CN.go"},
	}
	for _, tt := range tests {
		if got := IsNormalName(tt.name); got != tt.want {
			t.Errorf("IsNormalName(%q) = %v, want %v", tt.name, got, tt.want)
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

func TestOSArchTest(t *testing.T) {
	tests := []struct {
		name, goos, goarch string
		test               bool
	}{
		{"zsyscall_linux_s390x.go", "linux", "s390x", false},
		{"zsyscall_linux_x.go", "", "", false},
		{"doc_linux.go", "linux", "", false},
		{"atomic_pointer.go", "", "", false},
		{"doc_android_test.go", "android", "", true},
		{"doc_android_arm_test.go", "android", "arm", true},
		{"d_o_c_android_arm.go", "android", "arm", false},
	}
	for _, tt := range tests {
		if goos, goarch, test := OSArchTest(tt.name); goos != tt.goos ||
			goarch != tt.goarch || test != tt.test {
			t.Errorf("OSArchTest(%q) = %q,%q,%q want %q,%q,%q",
				goos, goarch, test,
				tt.goos, tt.goarch, tt.test)
		}
	}
}

func TestBuildForLinux(t *testing.T) {
	tests := []struct {
		want bool
		name string
	}{
		{false, ""},
		{false, "// no package"},
		{true, "package n"},
		{true, "//\npackage n"},
		{false, "// +build ingore\npackage n"},
		{false, "//+build ingore\npackage n"},
		{false, "// +build linux3 windows\npackage n"},
		{false, "// +buildlinux window\npackage n"},
		{false, "// +build !linux window\npackage n"},
		{true, "// +build linux window\npackage n"},
		{true, "// +build window linux\npackage n"},
	}
	for _, tt := range tests {
		if got := buildForLinux([]byte(tt.name)); got != tt.want {
			t.Errorf("buildForLinux(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
