package bootstrap

import (
	"fmt"

	"github.com/b4fun/parsimonious-go/types"
)

func asGrammar(v any, err error) (*types.Grammar, error) {
	if err != nil {
		return nil, err
	}

	if result, ok := v.(*types.Grammar); ok {
		return result, nil
	}
	return nil, fmt.Errorf("expected *Grammar, got %T", v)
}

func assertNodeToHaveChildrenCount(node *types.Node, children []any, count int) error {
	if len(node.Children) == count {
		return nil
	}

	return fmt.Errorf(
		"%s should have %d children, got %d",
		node, count, len(children),
	)
}

func shouldCastAsNode(v any) (*types.Node, error) {
	if node, ok := v.(*types.Node); ok {
		return node, nil
	} else {
		return nil, fmt.Errorf("expected *Node, got %#v", v)
	}
}

func shouldCastAsExpressionWithType[T types.Expression](v any) (T, error) {
	if expression, ok := v.(T); ok {
		return expression, nil
	} else {
		var empty T
		return empty, fmt.Errorf("expected Expression, got %#v", v)
	}
}

func shouldCastAsExpression(v any) (types.Expression, error) {
	if expression, ok := v.(types.Expression); ok {
		return expression, nil
	} else {
		return nil, fmt.Errorf("expected Expression, got %#v", v)
	}
}

func shouldCastAsExpressions(v any) ([]types.Expression, error) {
	exprs, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected []Expression, got %#v", v)
	}

	expressions := make([]types.Expression, len(exprs))
	for idx, expr := range exprs {
		if expression, ok := expr.(types.Expression); ok {
			expressions[idx] = expression
		} else {
			return nil, fmt.Errorf("expected Expression, got %#v", expr)
		}
	}

	return expressions, nil
}
