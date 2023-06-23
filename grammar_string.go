package parsimonious

import (
	"fmt"

	"github.com/dlclark/regexp2"
)

var pythonStringExpr, pythonStringVisitor = func() (Expression, *NodeVisitorMux) {
doubleQuotedCharacters := NewRegex(
		"",
		regexp2.MustCompile(
			`[^"\\]*(?:\\.[^"\\]*)*`,
			regexp2.RE2|regexp2.Singleline|regexp2.Unicode,
		),
	)
			singleQuotedCharacters := NewRegex(
				"",
				regexp2.MustCompile(
					`[^'\\]*(?:\\.[^'\\]*)*`,
					regexp2.RE2|regexp2.Singleline|regexp2.Unicode,
				),
			)

	doubleQuoted := NewSequence(
		"double_quoted",
		[]Expression{
			NewLiteral("\""),
			doubleQuotedCharacters,
			NewLiteral("\""),
		},
	)

	singleQuoted := NewSequence(
		"single_quoted",
		[]Expression{
			NewLiteral("'"),
			singleQuotedCharacters,
			NewLiteral("'"),
		},
	)

	rawStringDoubleQuoted := NewSequence(
		"raw_string_double_quoted",
		[]Expression{
			NewOneOf("", []Expression{NewLiteral("r"), NewLiteral("R")}),
			NewLiteral("\""),
			doubleQuotedCharacters,
			NewLiteral("\""),
		},
	)

	rawStringSingleQuoted := NewSequence(
		"raw_string_single_quoted",
		[]Expression{
			NewOneOf("", []Expression{NewLiteral("r"), NewLiteral("R")}),
			NewLiteral("'"),
			singleQuotedCharacters,
			NewLiteral("'"),
		},
	)

	stringValue := NewOneOf(
		"string_value",
		[]Expression{
			doubleQuoted,
			singleQuoted,
			rawStringDoubleQuoted,
			rawStringSingleQuoted,
		},
	)

	visitQuoted := func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		literalNode, err := shouldCastAsNode(children[1])
		if err != nil {
			return nil, err
		}
		rv := literalNode.Text

		return rv, nil
	}

	visitRawString := func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 4); err != nil {
			return nil, err
		}

		literalNode, err := shouldCastAsNode(children[2])
		if err != nil {
			return nil, err
		}
		rv := literalNode.Text

		return rv, nil
	}

	visitStringValue := func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 1); err != nil {
			return nil, err
		}

		return children[0], nil
	}

	mux := NewNodeVisitorMux(
		visitWithChildren(func(node *Node, children []any) (any, error) {
			if len(children) > 0 {
				return children, nil
			}
			return node, nil
		}),
	).
		VisitWithChildren("single_quoted", visitQuoted).
		VisitWithChildren("double_quoted", visitQuoted).
		VisitWithChildren("raw_string_double_quoted", visitRawString).
		VisitWithChildren("raw_string_single_quoted", visitRawString).
		VisitWithChildren("string_value", visitStringValue)

	return stringValue, mux
}()

func evalPythonStringValue(input string) (string, error) {
	tree, err := ParseWithExpression(pythonStringExpr, input, 0)
	if err != nil {
		return "", err
	}

	ss, err := pythonStringVisitor.Visit(tree)
	if err != nil {
		return "", err
	}

	s, ok := ss.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type %T", ss)
	}

	return s, nil
}
