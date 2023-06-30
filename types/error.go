package types

import (
	"fmt"
	"strings"
)

var (
	ErrLeftRecursionError = fmt.Errorf("left recursion error") // TODO: add position
)

type ErrParseFailed struct {
	Text       string
	Position   int
	Expression Expression
}

func newErrParseFailed(text string, position int, expression Expression) *ErrParseFailed {
	return &ErrParseFailed{
		Text:       text,
		Position:   position,
		Expression: expression,
	}
}

func (e *ErrParseFailed) Error() string {
	var ruleName string
	if e.Expression.ExprName() == "" {
		ruleName = e.Expression.String()
	} else {
		ruleName = fmt.Sprintf("%q", e.Expression.ExprName())
	}
	line, column := e.LineAndColumn()

	return fmt.Sprintf(
		"rule %s didn't match at %q (line %d, column %d)",
		ruleName,
		sliceStringAsRuneSliceWithLength(e.Text, e.Position, 20),
		line, column,
	)
}

func (e *ErrParseFailed) LineAndColumn() (int, int) {
	line := strings.Count(e.Text[:e.Position], "\n") + 1
	column := e.Position - strings.LastIndex(e.Text[:e.Position], "\n")

	return line, column
}

type ErrIncompleteParseFailed struct {
	ErrParseFailed
}

func newErrIncompleteParseFailed(
	text string,
	position int,
	expression Expression,
) *ErrIncompleteParseFailed {
	return &ErrIncompleteParseFailed{
		ErrParseFailed: *newErrParseFailed(text, position, expression),
	}
}

func (e *ErrIncompleteParseFailed) Error() string {
	line, column := e.LineAndColumn()
	return fmt.Sprintf(
		"rule %q matched in its entirely, but it didn't consume all the text. "+
			"The non-matching portion of the text begins with %q (line %d, column %d)",
		e.Expression.ExprName(),
		sliceStringAsRuneSliceWithLength(e.Text, e.Position, 20),
		line, column,
	)
}
