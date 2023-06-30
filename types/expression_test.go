package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertMatchAsNode(t testing.TB, expr Expression, text string, expected *Node) {
	node, err := expr.Match(text, createParseOpts())
	assert.NoError(t, err)
	assert.Equal(t, expected, node)
}

func Test_Expression_Match(t *testing.T) {
	t.Run("Literal", func(t *testing.T) {
		expr := NewLiteralWithName("greeting", "hello")
		assertMatchAsNode(
			t, expr, "hello",
			newNode(expr, "hello", 0, 5),
		)
	})

	t.Run("Sequence", func(t *testing.T) {
		expr := NewSequence(
			"dwarf",
			[]Expression{
				NewLiteral("heigh"),
				NewLiteral("ho"),
			},
		)
		text := "heighho"
		assertMatchAsNode(
			t, expr, text,
			newNodeWithChildren(
				expr,
				text, 0, 7,
				[]*Node{
					newNode(NewLiteral("heigh"), text, 0, 5),
					newNode(NewLiteral("ho"), text, 5, 7),
				},
			),
		)
	})
}
