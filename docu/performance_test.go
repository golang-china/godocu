package docu

import "testing"

func BenchmarkDocu_Parse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		du := New()
		du.Parse("go/types", nil)
	}
}
