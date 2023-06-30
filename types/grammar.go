package types

import "fmt"

// Grammar parses a text into a tree of nodes with defined grammar rules.
type Grammar struct {
	rules       map[string]Expression
	defaultRule Expression
}

// NewGrammar creates a new grammar with the given rules and default rule.
func NewGrammar(rules map[string]Expression, defaultRule Expression) *Grammar {
	return &Grammar{
		rules:       rules,
		defaultRule: defaultRule,
	}
}

func (g *Grammar) String() string {
	return fmt.Sprintf(
		"<Grammar #rules=%d defaultRule=%q>",
		len(g.rules),
		g.defaultRule.ExprName(),
	)
}

func (g *Grammar) Parse(text string, parseOpts ...ParseOption) (*Node, error) {
	return ParseWithExpression(g.defaultRule, text, parseOpts...)
}

func (g *Grammar) ParseWithRule(ruleName string, text string, parseOpts ...ParseOption) (*Node, error) {
	rule, ok := g.rules[ruleName]
	if !ok {
		return nil, fmt.Errorf("no such rule %q", ruleName)
	}
	return ParseWithExpression(rule, text, parseOpts...)
}

func (g *Grammar) GetRule(ruleName string) (Expression, bool) {
	rule, ok := g.rules[ruleName]
	return rule, ok
}
