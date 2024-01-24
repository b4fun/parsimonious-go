package property

import (
	"testing"

	"github.com/b4fun/parsimonious-go"
)

const grammarText = `
Item = "-" _ KeyValuePairs _

KeyValuePairs = 'item(' KeyValuePair ("," _ KeyValuePair)* ')'

KeyValuePair = Key _ "=" _ Value

Key = ~r"[a-zA-Z][a-zA-Z0-9_]*"

Value = String / Number / KeyValuePairs

String = StringLiteral / StringQuoted

StringLiteral = "string(" ~r'[^)]+' ")"
StringQuoted = "string(" _ '"' ~r'[^"]*' '"' _ ")"

Number = "number(" _ ~r"[0-9]+(\.[0-9]+)?" _ ")"

_ = Whitespace*

Whitespace = " " / "\t" / EOL

EOL = "\n" / "\r\n" / "\r"
`

func Test_Property(t *testing.T) {
	withDebug := parsimonious.ParseWithDebug(true)

	grammar, err := parsimonious.NewGrammar(grammarText, withDebug)
	if err != nil {
		t.Errorf("parse grammar failed: %v", err)
		return
	}

	program := `- item(name=string( Energy中文 ), subitem=item(value=number(997), unit=string("value")))`
	t.Logf("%q\n", program)

	tree, err := grammar.Parse(program, withDebug)
	if err != nil {
		t.Errorf("parse sample failed: %v", err)
		return
	}
	t.Log("\n" + parsimonious.DumpNodeExprTree(tree))
}