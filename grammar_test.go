package parsimonious

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Grammar_ParseErrors(t *testing.T) {

	t.Run("leftovers", func(t *testing.T) {
		grammar, err := NewGrammar(`seq = "a" (" " "b")+`)
		assert.NoError(t, err)

		tree, err := grammar.Parse("a bb")
		assert.Nil(t, tree)
		assert.Error(t, err)
		assert.IsType(t, &ErrIncompleteParseFailed{}, err)
		assert.Equal(
			t, err.Error(),
			`rule "seq" matched in its entirely, but it didn't consume all the text. `+
				`The non-matching portion of the text begins with "" (line 1, column 4)`,
		)
	})

	t.Run("left recursion", func(t *testing.T) {
		grammar, err := NewGrammar(`
expression = operator_expression / non_operator_expression
non_operator_expression = number_expression
operator_expression = expression "+" non_operator_expression
number_expression = ~"[0-9]+"
`,
		)
		assert.NoError(t, err)
		fmt.Println("here!!")

		tree, err := grammar.ParseWithRule(
			"operator_expression",
			"1+2",
			ParseWithDebug(true),
		)
		assert.Nil(t, tree)
		assert.Error(t, err)
		assert.IsType(t, &ErrLeftRecursion{}, err)
		assert.Equal(
			t, err.Error(),
			"",
		)
	})
}
