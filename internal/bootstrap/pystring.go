package bootstrap

import (
	"fmt"

	"github.com/b4fun/parsimonious-go/nodes"
	"github.com/b4fun/parsimonious-go/types"
	"github.com/dlclark/regexp2"
)

var pythonStringExpr, pythonStringVisitor = func() (types.Expression, *nodes.NodeVisitorMux) {
	doubleQuotedCharacters := types.NewRegex(
		"",
		regexp2.MustCompile(
			`[^"\\]*(?:\\.[^"\\]*)*`,
			regexp2.RE2|regexp2.Singleline|regexp2.Unicode,
		),
	)
	singleQuotedCharacters := types.NewRegex(
		"",
		regexp2.MustCompile(
			`[^'\\]*(?:\\.[^'\\]*)*`,
			regexp2.RE2|regexp2.Singleline|regexp2.Unicode,
		),
	)

	doubleQuoted := types.NewSequence(
		"double_quoted",
		[]types.Expression{
			types.NewLiteral("\""),
			doubleQuotedCharacters,
			types.NewLiteral("\""),
		},
	)

	singleQuoted := types.NewSequence(
		"single_quoted",
		[]types.Expression{
			types.NewLiteral("'"),
			singleQuotedCharacters,
			types.NewLiteral("'"),
		},
	)

	representPrefix := types.NewOneOf(
		"",
		[]types.Expression{
			types.NewLiteral("r"),
			types.NewLiteral("R"),
		},
	)

	rawStringDoubleQuoted := types.NewSequence(
		"raw_string_double_quoted",
		[]types.Expression{
			representPrefix,
			types.NewLiteral("\""),
			doubleQuotedCharacters,
			types.NewLiteral("\""),
		},
	)

	rawStringSingleQuoted := types.NewSequence(
		"raw_string_single_quoted",
		[]types.Expression{
			representPrefix,
			types.NewLiteral("'"),
			singleQuotedCharacters,
			types.NewLiteral("'"),
		},
	)

	stringValue := types.NewOneOf(
		"string_value",
		[]types.Expression{
			doubleQuoted,
			singleQuoted,
			rawStringDoubleQuoted,
			rawStringSingleQuoted,
		},
	)

	visitQuoted := func(node *types.Node, children []any) (any, error) {
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

	visitRawString := func(node *types.Node, children []any) (any, error) {
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

	visitStringValue := func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 1); err != nil {
			return nil, err
		}

		return children[0], nil
	}

	mux := nodes.NewNodeVisitorMux().
		HandleExpr("single_quoted", visitQuoted).
		HandleExpr("double_quoted", visitQuoted).
		HandleExpr("raw_string_double_quoted", visitRawString).
		HandleExpr("raw_string_single_quoted", visitRawString).
		HandleExpr("string_value", visitStringValue)

	return stringValue, mux
}()

func evalPythonStringValue(input string) (string, error) {
	tree, err := types.ParseWithExpression(pythonStringExpr, input)
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
