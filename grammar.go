package parsimonious

import (
	"fmt"

	"github.com/dlclark/regexp2"
)

func asGrammar(v any, err error) (*Grammar, error) {
	if err != nil {
		return nil, err
	}

	if result, ok := v.(*Grammar); ok {
		return result, nil
	}
	return nil, fmt.Errorf("expected *Grammar, got %T", v)
}

func shouldCastAsNode(v any) (*Node, error) {
	if node, ok := v.(*Node); ok {
		return node, nil
	} else {
		return nil, fmt.Errorf("expected *Node, got %#v", v)
	}
}

func shouldCastAsExpressionWithType[T Expression](v any) (T, error) {
	if expression, ok := v.(T); ok {
		return expression, nil
	} else {
		var empty T
		return empty, fmt.Errorf("expected Expression, got %#v", v)
	}
}

func shouldCastAsExpression(v any) (Expression, error) {
	if expression, ok := v.(Expression); ok {
		return expression, nil
	} else {
		return nil, fmt.Errorf("expected Expression, got %#v", v)
	}
}

func shouldCastAsExpressions(v any) ([]Expression, error) {
	exprs, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected []Expression, got %#v", v)
	}

	expressions := make([]Expression, len(exprs))
	for idx, expr := range exprs {
		if expression, ok := expr.(Expression); ok {
			expressions[idx] = expression
		} else {
			return nil, fmt.Errorf("expected Expression, got %#v", expr)
		}
	}

	return expressions, nil
}

var parsimoniousGrammar  *Grammar

var spacelessLiteral =NewOneOf(
				"spaceless_literal",
				[]Expression{
					NewRegex(
						"",
						regexp2.MustCompile(`(?si)^r?"[^"\\]*(?:\\.[^"\\]*)*"`, regexp2.RE2),
					),
					NewRegex(
						"",
						regexp2.MustCompile(`(?si)^r?'[^'\\]*(?:\\.[^'\\]*)*'`, regexp2.RE2),
					),
				},
			)

func initParsimoniousGrammar() (*Grammar, error) {
	mux:= createRuleVisitor(false, []Expression{spacelessLiteral})
	bootstrapTree, err := ParseWithExpression(createBootstrapRules(), ruleSyntax, 0)
	if err != nil {
		return nil, fmt.Errorf("parse bootstrap grammar: %w", err)
	}

	bootstrapGrammar, err := asGrammar(mux.Visit(bootstrapTree))
	if err != nil {
		return nil, fmt.Errorf("visit bootstrap grammar: %w", err)
	}

	tree, err := bootstrapGrammar.Parse(ruleSyntax)
	if err != nil {
		return nil, fmt.Errorf("parse parsimonious grammar: %w", err)
	}


	result, err := asGrammar(mux.Visit(tree))
	if err != nil {
		return nil, fmt.Errorf("visit parsimonious grammar: %w", err)
	}

	return result, nil
}

func init() {
	var err error
	parsimoniousGrammar, err = initParsimoniousGrammar()
	if err != nil {
		panic(fmt.Errorf("init parsimonious grammar: %w", err))
	}
}

type Grammar struct {
	rules map[string]Expression
	defaultRule Expression
}

func (g *Grammar) String() string {
	return fmt.Sprintf(
		"<Grammar #rules=%d defaultRule=%q>",
		len(g.rules),
		g.defaultRule.ExprName(),
	)
}

func (g *Grammar) Parse(text string) (*Node, error) {
	t, err := g.defaultRule.Match(text, 0)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func (g *Grammar) ParseWithRule(ruleName string, text string) (*Node, error) {
	rule, ok := g.rules[ruleName]
	if !ok {
		return nil, fmt.Errorf("no such rule %q", ruleName)
	}
	return rule.Match(text, 0)
}

func (g *Grammar) GetRule(ruleName string) (Expression, bool) {
	rule, ok := g.rules[ruleName]
	return rule, ok
}

func NewGrammar(input string) (*Grammar, error) {
	tree, err := parsimoniousGrammar.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("parse grammar: %w", err)
	}

	mux := createRuleVisitor(false, nil)
	return asGrammar(mux.Visit(tree))
}