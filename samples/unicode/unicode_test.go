package unicode

import (
	"testing"

	"github.com/b4fun/parsimonious-go"
)


const grammar = `
#指令 = 空格* (注释 | 语句) 空格* 换行?
#换行 = "\n" / "\r\n"
#空格 = " " / "\t"
#注释 = "#" (!换行 .)*
#语句 = 数字* 空格* 操作符 空格* 数字*
#数字 = ~r"[0-9]"
#操作符 = "+" / "-"

中文 = "a" / "b" / "c"
`

func Test_UnicodeGrammar(t *testing.T) {
	grammar, err := parsimonious.NewGrammar(grammar)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := grammar.Parse("1+2")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tree)
}