package docu

import "testing"

func TestWrapComments(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"", ""}, {"     ", ""}, {"\n", ""}, {"\t\n", ""},
		{
			"the mode, using the given Block. The length of iv must be the same as the Block's",
			"// the mode, using the given Block. The length of iv must be the same as the\n// Block's\n",
		},
		{
			"InsertAfter inserts a new element e with value v immediately after mark and returns e.",
			"// InsertAfter inserts a new element e with value v immediately after mark and\n// returns e.\n",
		},
		{
			"Pop removes the minimum element (according to Less) from the heap and returns it.",
			"// Pop removes the minimum element (according to Less) from the heap and returns\n// it.\n",
		},
		{
			"注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆添加或者删除元素。",
			"// 注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆\n// 添加或者删除元素。\n",
		},
		{
			"tab\n\tcode\n\n注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆添加或者删除元素。",
			"// tab\n//\n// \tcode\n//\n// 注意接口的Push和Pop方法是供heap包调用的，请使用heap.Push和heap.Pop来向一个堆\n// 添加或者删除元素。\n",
		},
		{
			"将一个Stream与一个io.Writer接口关联起来，Write方法会调用XORKeyStream方法来处理提供的所有切片。如果Write方法返回的n小于提供的切片的长度，则表示StreamWriter不同步，必须丢弃。StreamWriter没有内建的缓存，不需要调用Close方法去清空缓存。",
			"// 将一个Stream与一个io.Writer接口关联起来，Write方法会调用XORKeyStream方法来处\n// 理提供的所有切片。如果Write方法返回的n小于提供的切片的长度，则表示\n" +
				"// StreamWriter不同步，必须丢弃。StreamWriter没有内建的缓存，不需要调用Close方法\n// 去清空缓存。\n",
		},
		{
			"SignPSS采用RSASSA-PSS方案计算签名。注意hashed必须是使用提供给本函数的hash参数对（要签名的）原始数据进行hash的结果。opts参数可以为nil，此时会使用默认参数。\n",
			"// SignPSS采用RSASSA-PSS方案计算签名。注意hashed必须是使用提供给本函数的hash参数\n// 对（要签名的）原始数据进行hash的结果。opts参数可以为nil，此时会使用默认参数。\n",
		},
		{
			"一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五六七八。九十",
			"// 一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五六七\n// 八。九十\n",
		},
		{
			"http://123.456.com/3243214324234354354353245345435435435435435435435435342543543 yes",
			"// http://123.456.com/3243214324234354354353245345435435435435435435435435342543543\n// yes\n",
		},
		{
			"see http://123.456.com/3243214324234354354353245345435435435435435435435435342543543",
			"// see\n// http://123.456.com/3243214324234354354353245345435435435435435435435435342543543\n",
		},
		{
			"see http://123.456.com/3243214324234354354353245345435435435435435435435433333 reame.me",
			"// see\n// http://123.456.com/3243214324234354354353245345435435435435435435435433333\n// reame.me\n",
		},
		{
			"see\n    http://123.456.com/3243214324234354354353245345435435435435435435435433333 reame.me",
			"// see\n//\n// \thttp://123.456.com/3243214324234354354353245345435435435435435435435433333 reame.me\n",
		},
	}
	for _, tt := range tests {
		if got := WrapComments(tt.text, "// ", 77); got != tt.want {
			t.Errorf("WrapComments(%q) =\n%q\nwant\n%q", tt.text, got, tt.want)
		}
	}
}
