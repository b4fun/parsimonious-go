package parsimonious

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Grammar_ParseErrors(t *testing.T) {
	t.Run("leftovers", func(t *testing.T) {
		grammar, err := NewGrammar(`seq = "a" (" " "b")+`)
		assert.NoError(t, err)

		tree, err := grammar.Parse("a bb")
		assert.Nil(t, tree)
		assert.IsType(t, &ErrIncompleteParseFailed{}, err)
		assert.Equal(
			t, err.Error(),
			`rule "seq" matched in its entirely, but it didn't consume all the text. The non-matching portion of the text begins with "" (line 1, column 4)`,
		)
	})
}