package parsimonious

import (
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
)

// ruleSyntax is the syntax of the parsimonious grammar.
const ruleSyntax = `
# Ignored things (represented by _) are typically hung off the end of the
# leafmost kinds of nodes. Literals like "/" count as leaves.

rules = _ rule*
rule = label equals expression
equals = "=" _
literal = spaceless_literal _

# FIXME(hbc): invalid regex
# spaceless_literal = ~"r?\"[^\"\\\\]*(?:\\\\.[^\"\\\\]*)*\""is /
# 					  ~"r?'[^'\\\\]*(?:\\\\.[^'\\\\]*)*'"is

expression = ored / sequence / term
or_term = "/" _ term
ored = term or_term+
sequence = term term+
not_term = "!" term _
lookahead_term = "&" term _
term = not_term / lookahead_term / quantified / atom
quantified = atom quantifier
atom = reference / literal / regex / parenthesized
regex = "~" spaceless_literal ~"[ilmsuxa]*"i _
parenthesized = "(" _ expression ")" _
quantifier = ~r"[*+?]|\{\d*,\d+\}|\{\d+,\d*\}|\{\d+\}" _
reference = label !equals

# A subsequent equal sign is the only thing that distinguishes a label
# (which begins a new rule) from a reference (which is just a pointer to a
# rule defined somewhere else):
label = ~"[a-zA-Z_][a-zA-Z_0-9]*(?![\"'])" _

# _ = ~"\\s*(?:#[^\\r\\n]*)?\\s*"
_ = meaninglessness*
meaninglessness = ~r"\s+" / comment
comment = ~r"#[^\r\n]*"
`

// createBootstrapRules returns the bootstrap rules for the parsimonious grammar.
func createBootstrapRules() Expression {
	comment := NewRegex(
		"comment",
		regexp2.MustCompile("^#[^\r\n]*", regexp2.RE2),
	)
	meaninglessness := NewOneOf(
		"meaninglessness",
		[]Expression{
			NewRegex("", regexp2.MustCompile(`^\s+`, regexp2.RE2)),
			comment,
		},
	)
	underscore := NewZeroOrMore(
		"_",
		meaninglessness,
	)
	equals := NewSequence(
		"equals",
		[]Expression{
			NewLiteral("="),
			underscore,
		},
	)
	label := NewSequence(
		"label",
		[]Expression{
			NewRegex("", regexp2.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`, regexp2.RE2)),
			underscore,
		},
	)
	reference := NewSequence(
		"reference",
		[]Expression{
			label,
			NewNot(equals),
		},
	)
	quantifier := NewSequence(
		"quantifier",
		[]Expression{
			NewRegex("", regexp2.MustCompile(`^[*+?]`, regexp2.RE2)),
			underscore,
		},
	)
	spacelessLiteral := NewRegex(
		"spaceless_literal",
		regexp2.MustCompile(`(?si)^r?"[^"\\]*(?:\\.[^"\\]*)*"`, regexp2.RE2),
	)
	literal := NewSequence(
		"literal",
		[]Expression{
			spacelessLiteral,
			underscore,
		},
	)
	regex := NewSequence(
		"regex",
		[]Expression{
			NewLiteral("~"),
			literal,
			NewRegex("", regexp2.MustCompile(`^[ilmsuxa]*`, regexp2.RE2|regexp2.IgnoreCase)),
			underscore,
		},
	)
	atom := NewOneOf(
		"atom",
		[]Expression{
			reference,
			literal,
			regex,
		},
	)
	quantified := NewSequence(
		"quantified",
		[]Expression{
			atom,
			quantifier,
		},
	)

	term := NewOneOf(
		"term",
		[]Expression{
			quantified,
			atom,
		},
	)
	notTerm := NewSequence(
		"not_term",
		[]Expression{
			NewLiteral("!"),
			term,
			underscore,
		},
	)
	term.members = []Expression{
		notTerm,
		quantified,
		atom,
	}

	sequence := NewSequence(
		"sequence",
		[]Expression{
			term,
			NewOneOrMore("", term),
		},
	)
	orTerm := NewSequence(
		"or_term",
		[]Expression{
			NewLiteral("/"),
			underscore,
			term,
		},
	)
	ored := NewSequence(
		"ored",
		[]Expression{
			term,
			NewOneOrMore("", orTerm),
		},
	)
	expression := NewOneOf(
		"expression",
		[]Expression{
			ored,
			sequence,
			term,
		},
	)
	rule := NewSequence(
		"rule",
		[]Expression{
			label,
			equals,
			expression,
		},
	)
	rules := NewSequence(
		"rules",
		[]Expression{
			underscore,
			NewOneOrMore("", rule),
		},
	)

	return rules
}

// createRuleVisitor creates a node visitor for the parsimonious grammar rules.
func createRuleVisitor(
	debug bool,
	customRules []Expression,
) *NodeVisitorMux {
	debugf := func(s string, args ...any) {
		if debug {
			fmt.Printf("[rule visitor] "+s, args...)
		}
	}

	debugVisitWithChildren := func(f NodeVisitWithChildrenFunc) NodeVisitWithChildrenFunc {
		return func(node *Node, children []any) (any, error) {
			debugf(
				"[%s visitor] visiting %s with children (count=%d)\n",
				node.Expression.ExprName(), node, len(children),
			)

			return f(node, children)
		}
	}

	defaultVisitor := visitWithChildren(func(node *Node, children []any) (any, error) {
		debugf(
			"[default visitor] visiting <Node: %s start:%d, end:%d> with children (count=%d)\n",
			node.Expression, node.Start, node.End, len(children),
		)

		if len(children) > 0 {
			return children, nil
		}
		return node, nil
	})

	liftChild := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if len(children) < 1 {
			return nil, fmt.Errorf("%s should have at least one child", node)
		}

		return children[0], nil
	})

	visitParenthesized := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 5); err != nil {
			return nil, err
		}

		expression, err := shouldCastAsExpression(children[2])
		if err != nil {
			return nil, fmt.Errorf("parenthesized: %w", err)
		}

		return expression, nil
	})

	visitQuantifier := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		symbol := children[0]
		return symbol, nil
	})

	visitQuantified := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		atom, err := shouldCastAsExpression(children[0])
		if err != nil {
			return nil, fmt.Errorf("quantified: %w", err)
		}
		quantifier, err := shouldCastAsNode(children[1])
		if err != nil {
			return nil, fmt.Errorf("quantified: %w", err)
		}

		switch t := quantifier.Text; t {
		case "?":
			return NewOptional("", atom), nil
		case "*":
			return NewZeroOrMore("", atom), nil
		case "+":
			return NewOneOrMore("", atom), nil
		default:
			// TODO: support quantifiers like {1,2}
			return nil, fmt.Errorf("TODO: support quantifier %q", t)
		}
	})

	visitLookaheadTerm := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		term, err := shouldCastAsExpression(children[1])
		if err != nil {
			return nil, fmt.Errorf("lookahead_term: %w", err)
		}

		return NewLookahead("", term, false), nil
	})

	visitNotTerm := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		term, err := shouldCastAsExpression(children[1])
		if err != nil {
			return nil, fmt.Errorf("not_term: %w", err)
		}

		return NewNot(term), nil
	})

	visitRule := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		label, err := shouldCastAsNode(children[0])
		if err != nil {
			return nil, fmt.Errorf("rule: %w", err)
		}
		expression, err := shouldCastAsExpression(children[2])
		if err != nil {
			return nil, fmt.Errorf("rule: %w", err)
		}

		debugf("setting rule name %q to %s\n", label.Text, expression)
		expression.SetExprName(label.Text)

		return expression, nil
	})

	visitSequence := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		term, err := shouldCastAsExpression(children[0])
		if err != nil {
			return nil, fmt.Errorf("sequence: %w", err)
		}
		otherTerms, err := shouldCastAsExpressions(children[1])
		if err != nil {
			return nil, fmt.Errorf("sequence: %w", err)
		}

		sequenceMembers := append([]Expression{term}, otherTerms...)
		debugf("creating sequence members with length %d\n", len(sequenceMembers))
		return NewSequence("", sequenceMembers), nil
	})

	visitOred := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		firstTerm, err := shouldCastAsExpression(children[0])
		if err != nil {
			return nil, fmt.Errorf("ored: %w", err)
		}
		otherTerms, err := shouldCastAsExpressions(children[1])
		if err != nil {
			return nil, fmt.Errorf("ored: %w", err)
		}

		terms := append([]Expression{firstTerm}, otherTerms...)
		debugf("creating oneOf members with length %d\n", len(terms))
		return NewOneOf("", terms), nil
	})

	visitOrTerm := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		return children[2], nil
	})

	// FIXME: this visitor is returning non expression value, which makes us to fallback to any :(
	visitLabel := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		labelName, err := shouldCastAsNode(children[0])
		if err != nil {
			return nil, err
		}
		return labelName, nil
	})

	visitReference := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		label, err := shouldCastAsNode(children[0])
		if err != nil {
			return nil, err
		}
		return NewLazyReference(label.Text), nil
	})

	visitRegex := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 4); err != nil {
			return nil, err
		}

		literal, err := shouldCastAsExpressionWithType[*Literal](children[1])
		if err != nil {
			return nil, fmt.Errorf("regex (literal): %w", err)
		}
		pattern := "^" + literal.literal

		var reOptions regexp2.RegexOptions = regexp2.RE2
		flags, err := shouldCastAsNode(children[2])
		if err != nil {
			return nil, fmt.Errorf("regex (flags): %w", err)
		}
		flagsText := strings.ToLower(flags.Text)
		if strings.Contains(flagsText, "i") {
			reOptions |= regexp2.IgnoreCase
		}
		if strings.Contains(flagsText, "l") {
			return nil, fmt.Errorf("regex (flags): flag 'l' is not supported")
		}
		if strings.Contains(flagsText, "m") {
			reOptions |= regexp2.Multiline
		}
		if strings.Contains(flagsText, "s") {
			reOptions |= regexp2.Singleline
		}

		re, err := regexp2.Compile(pattern, reOptions)
		if err != nil {
			return nil, fmt.Errorf("regex: %q %w", pattern, err)
		}

		debugf("regex pattern: %q, flags: %q\n", pattern, flagsText)
		return NewRegex("", re), nil
	})

	visitSpacelessLiteral := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		literalValue, err := evalPythonStringValue(node.Text)
		if err != nil {
			return nil, fmt.Errorf("spaceless literal: %q %w", node.Text, err)
		}

		return NewLiteral(literalValue), nil
	})

	visitLiteral := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		return children[0], nil
	})

	// FIXME: this visitor is returning non expression value, which makes us to fallback to any :(
	visitRules := debugVisitWithChildren(func(node *Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		rules, err := shouldCastAsExpressions(children[1])
		if err != nil {
			return nil, fmt.Errorf("rules: %w", err)
		}

		rulesMap := make(map[string]Expression)
		for _, rule := range rules {
			rulesMap[rule.ExprName()] = rule
		}
		for _, rule := range customRules {
			rulesMap[rule.ExprName()] = rule
		}
		for k, v := range rulesMap {
			withResolveRefs, ok := v.(WithResolveRefs)
			if !ok {
				continue
			}
			resolved, err := withResolveRefs.ResolveRefs(rulesMap)
			if err != nil {
				return nil, fmt.Errorf("resolve refs for %q: %w", k, err)
			}
			rulesMap[k] = resolved
		}

		rv := &Grammar{
			rules:       rulesMap,
			defaultRule: rulesMap[rules[0].ExprName()],
		}
		debugf("loaded %d rules, default rule: %s\n", len(rv.rules), rv.defaultRule)

		return rv, nil
	})

	mux := NewNodeVisitorMux(defaultVisitor).
		VisitWithChildren("expression", liftChild).
		VisitWithChildren("term", liftChild).
		VisitWithChildren("atom", liftChild).
		VisitWithChildren("parenthesized", visitParenthesized).
		VisitWithChildren("quantifier", visitQuantifier).
		VisitWithChildren("quantified", visitQuantified).
		VisitWithChildren("lookahead_term", visitLookaheadTerm).
		VisitWithChildren("not_term", visitNotTerm).
		VisitWithChildren("rule", visitRule).
		VisitWithChildren("sequence", visitSequence).
		VisitWithChildren("ored", visitOred).
		VisitWithChildren("or_term", visitOrTerm).
		VisitWithChildren("label", visitLabel).
		VisitWithChildren("reference", visitReference).
		VisitWithChildren("regex", visitRegex).
		VisitWithChildren("spaceless_literal", visitSpacelessLiteral).
		VisitWithChildren("literal", visitLiteral).
		VisitWithChildren("rules", visitRules)

	return mux
}
