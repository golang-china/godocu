package docu

import "testing"

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		want int
		text string
	}{
		{79, "// 注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆"},
	}
	for _, tt := range tests {
		if got := visualWidth(tt.text); got != tt.want {
			t.Errorf("visualWidth() =\n%v\nwant\n%v", got, tt.want)
		}
	}
}

func TestLineWrapper(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{
			"the mode, using the given Block. The length of iv must be the same as the Block's",
			"// the mode, using the given Block. The length of iv must be the same as the\n// Block's",
		},
		{
			"InsertAfter inserts a new element e with value v immediately after mark and returns e.",
			"// InsertAfter inserts a new element e with value v immediately after mark and\n// returns e.",
		},
		{
			"Pop removes the minimum element (according to Less) from the heap and returns it.",
			"// Pop removes the minimum element (according to Less) from the heap and returns\n// it.",
		},
		{
			"注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆添加或者删除元素。",
			"// 注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆\n// 添加或者删除元素。",
		},
		{
			"tab\n	code\n\n注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆添加或者删除元素。",
			"// tab\n//     code\n//\n// 注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆\n// 添加或者删除元素。",
		},
		{
			"将一个Stream与一个io.Writer接口关联起来，Write方法会调用XORKeyStream方法来处理提供的所有切片。如果Write方法返回的n小于提供的切片的长度，则表示StreamWriter不同步，必须丢弃。StreamWriter没有内建的缓存，不需要调用Close方法去清空缓存。",
			"// 将一个Stream与一个io.Writer接口关联起来，Write方法会调用XORKeyStream方法来处\n// 理提供的所有切片。如果Write方法返回的n小于提供的切片的长度，则表示\n" +
				"// StreamWriter不同步，必须丢弃。StreamWriter没有内建的缓存，不需要调用Close方法\n// 去清空缓存。",
		},
		{
			"SignPSS采用RSASSA-PSS方案计算签名。注意hashed必须是使用提供给本函数的hash参数对（要签名的）原始数据进行hash的结果。opts参数可以为nil，此时会使用默认参数。\n",
			"// SignPSS采用RSASSA-PSS方案计算签名。注意hashed必须是使用提供给本函数的hash参数\n// 对（要签名的）原始数据进行hash的结果。opts参数可以为nil，此时会使用默认参数。",
		},
	}
	for _, tt := range tests {
		if got := LineWrapper(tt.text, "// ", 77); got != tt.want+"\n" {
			t.Errorf("LineWrapper() =\n%v\nwant\n%v", got, tt.want)
		}
	}
}
