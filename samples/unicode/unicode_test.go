package unicode

import (
	"testing"

	"github.com/b4fun/parsimonious-go"
)

const grammarText = `
program = _ statement*
statement = digits (operator digits)*
digits = digit+ _
digit = "0️⃣" / "1️⃣" / "2️⃣" / "3️⃣" / "4️⃣" / "5️⃣" / "6️⃣" / "7️⃣" / "8️⃣" / "9️⃣"
operator = ("➕" / "➖") _
_ = meaninglessness*
meaninglessness = ~r"\s+" / comment
comment = ~r"#[^\r\n]*"
`

func Test_UnicodeGrammar(t *testing.T) {
	withDebug := parsimonious.ParseWithDebug(true)

	grammar, err := parsimonious.NewGrammar(grammarText, withDebug)
	if err != nil {
		t.Errorf("parse grammar failed: %v", err)
		return
	}

	program := `
	# comment - 1

	0️⃣ ➕ 1️⃣
	
# comment - 2

9️⃣4️⃣7️⃣ ➕ 1️⃣ ➖ 2️⃣	➕ 3️⃣4️⃣
	`
	t.Logf("%q\n", program)

	tree, err := grammar.Parse(program, withDebug)
	if err != nil {
		t.Errorf("parse sample failed: %v", err)
		return
	}
	t.Log("\n" + parsimonious.DumpNodeExprTree(tree))

	countStatements := 0
	countDigits := 0
	mux := parsimonious.NewNodeVisitorMux(
		parsimonious.VisitWithChildren(func(node *parsimonious.Node, children []interface{}) (interface{}, error) {
			t.Logf("visiting node with default visitor: %s", node)

			return node.Text, nil
		}),
	).
		VisitWithChildren("statement", func(node *parsimonious.Node, children []interface{}) (interface{}, error) {
			countStatements++

			return children, nil
		}).
		VisitWithChildren("digits", func(node *parsimonious.Node, children []interface{}) (interface{}, error) {
			countDigits++

			return children, nil
		})
	_, err = mux.Visit(tree)
	if err != nil {
		t.Errorf("mux visit error: %v", err)
		return
	}

	if countStatements != 2 {
		t.Errorf("expect 2 statements, got %d", countStatements)
		return
	}
	if countDigits != 6 {
		t.Errorf("expect 6 digits, got %d", countDigits)
		return
	}
}
