package bootstrap

import (
	"fmt"
	"strings"

	"github.com/b4fun/parsimonious-go/nodes"
	"github.com/b4fun/parsimonious-go/types"
	"github.com/dlclark/regexp2"
)

// createBootstrapRules returns the bootstrap rules for the parsimonious grammar.
func createBootstrapRules() types.Expression {
	comment := types.NewRegex(
		"comment",
		regexp2.MustCompile("^#[^\r\n]*", regexp2.RE2),
	)
	meaninglessness := types.NewOneOf(
		"meaninglessness",
		[]types.Expression{
			types.NewRegex("", regexp2.MustCompile(`^\s+`, regexp2.RE2)),
			comment,
		},
	)
	underscore := types.NewZeroOrMore(
		"_",
		meaninglessness,
	)
	equals := types.NewSequence(
		"equals",
		[]types.Expression{
			types.NewLiteral("="),
			underscore,
		},
	)
	label := types.NewSequence(
		"label",
		[]types.Expression{
			types.NewRegex("", regexp2.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*`, regexp2.RE2)),
			underscore,
		},
	)
	reference := types.NewSequence(
		"reference",
		[]types.Expression{
			label,
			types.NewNot(equals),
		},
	)
	quantifier := types.NewSequence(
		"quantifier",
		[]types.Expression{
			types.NewRegex("", regexp2.MustCompile(`^[*+?]`, regexp2.RE2)),
			underscore,
		},
	)
	spacelessLiteral := types.NewRegex(
		"spaceless_literal",
		regexp2.MustCompile(`(?si)^r?"[^"\\]*(?:\\.[^"\\]*)*"`, regexp2.RE2),
	)
	literal := types.NewSequence(
		"literal",
		[]types.Expression{
			spacelessLiteral,
			underscore,
		},
	)
	regex := types.NewSequence(
		"regex",
		[]types.Expression{
			types.NewLiteral("~"),
			literal,
			types.NewRegex("", regexp2.MustCompile(`^[ilmsuxa]*`, regexp2.RE2|regexp2.IgnoreCase)),
			underscore,
		},
	)
	atom := types.NewOneOf(
		"atom",
		[]types.Expression{
			reference,
			literal,
			regex,
		},
	)
	quantified := types.NewSequence(
		"quantified",
		[]types.Expression{
			atom,
			quantifier,
		},
	)

	term := types.NewOneOf(
		"term",
		[]types.Expression{
			quantified,
			atom,
		},
	)
	notTerm := types.NewSequence(
		"not_term",
		[]types.Expression{
			types.NewLiteral("!"),
			term,
			underscore,
		},
	)
	term.SetMembers([]types.Expression{
		notTerm,
		quantified,
		atom,
	})

	sequence := types.NewSequence(
		"sequence",
		[]types.Expression{
			term,
			types.NewOneOrMore("", term),
		},
	)
	orTerm := types.NewSequence(
		"or_term",
		[]types.Expression{
			types.NewLiteral("/"),
			underscore,
			term,
		},
	)
	ored := types.NewSequence(
		"ored",
		[]types.Expression{
			term,
			types.NewOneOrMore("", orTerm),
		},
	)
	expression := types.NewOneOf(
		"expression",
		[]types.Expression{
			ored,
			sequence,
			term,
		},
	)
	rule := types.NewSequence(
		"rule",
		[]types.Expression{
			label,
			equals,
			expression,
		},
	)
	rules := types.NewSequence(
		"rules",
		[]types.Expression{
			underscore,
			types.NewOneOrMore("", rule),
		},
	)

	return rules
}

// createRuleVisitor creates a node visitor for the parsimonious grammar rules.
func createRuleVisitor(
	debug bool,
	customRules []types.Expression,
) *nodes.NodeVisitorMux {
	debugf := func(s string, args ...any) {
		if debug {
			fmt.Printf("[rule visitor] "+s, args...)
		}
	}

	debugHandleExpr := func(f nodes.NodeVisitFunc) nodes.NodeVisitFunc {
		return func(node *types.Node, children []any) (any, error) {
			debugf(
				"[%s visitor] visiting %s with children (count=%d)\n",
				node.Expression.ExprName(), node, len(children),
			)

			return f(node, children)
		}
	}

	defaultVisitorWithDebug := func(node *types.Node, children []any) (any, error) {
		debugf(
			"[default visitor] visiting <Node: %s start:%d, end:%d> with children (count=%d)\n",
			node.Expression, node.Start, node.End, len(children),
		)

		return nodes.DefaultNodeVisitor(node, children)
	}

	liftChild := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if len(children) < 1 {
			return nil, fmt.Errorf("%s should have at least one child", node)
		}

		return children[0], nil
	})

	visitParenthesized := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 5); err != nil {
			return nil, err
		}

		expression, err := shouldCastAsExpression(children[2])
		if err != nil {
			return nil, fmt.Errorf("parenthesized: %w", err)
		}

		return expression, nil
	})

	visitQuantifier := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		symbol := children[0]
		return symbol, nil
	})

	visitQuantified := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
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
			return types.NewOptional("", atom), nil
		case "*":
			return types.NewZeroOrMore("", atom), nil
		case "+":
			return types.NewOneOrMore("", atom), nil
		default:
			// TODO: support quantifiers like {1,2}
			return nil, fmt.Errorf("TODO: support quantifier %q", t)
		}
	})

	visitLookaheadTerm := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		term, err := shouldCastAsExpression(children[1])
		if err != nil {
			return nil, fmt.Errorf("lookahead_term: %w", err)
		}

		return types.NewLookahead("", term, false), nil
	})

	visitNotTerm := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		term, err := shouldCastAsExpression(children[1])
		if err != nil {
			return nil, fmt.Errorf("not_term: %w", err)
		}

		return types.NewNot(term), nil
	})

	visitRule := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
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

	visitSequence := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
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

		sequenceMembers := append([]types.Expression{term}, otherTerms...)
		debugf("creating sequence members with length %d\n", len(sequenceMembers))
		return types.NewSequence("", sequenceMembers), nil
	})

	visitOred := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
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

		terms := append([]types.Expression{firstTerm}, otherTerms...)
		debugf("creating oneOf members with length %d\n", len(terms))
		return types.NewOneOf("", terms), nil
	})

	visitOrTerm := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 3); err != nil {
			return nil, err
		}

		return children[2], nil
	})

	// FIXME: this visitor is returning non expression value, which makes us to fallback to any :(
	visitLabel := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		labelName, err := shouldCastAsNode(children[0])
		if err != nil {
			return nil, err
		}
		return labelName, nil
	})

	visitReference := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		label, err := shouldCastAsNode(children[0])
		if err != nil {
			return nil, err
		}
		return types.NewLazyReference(label.Text), nil
	})

	visitRegex := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 4); err != nil {
			return nil, err
		}

		literal, err := shouldCastAsExpressionWithType[*types.Literal](children[1])
		if err != nil {
			return nil, fmt.Errorf("regex (literal): %w", err)
		}
		pattern := "^" + literal.GetLiteral()

		var reOptions regexp2.RegexOptions = regexp2.Unicode
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
		return types.NewRegex("", re), nil
	})

	visitSpacelessLiteral := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		//debugf("spaceless literal: %q\n", node.Text)
		literalValue, err := evalPythonStringValue(node.Text)
		if err != nil {
			//debugf("spaceless literal %q eval failed %s\n", node.Text, err)
			return nil, fmt.Errorf("spaceless literal: %q %w", node.Text, err)
		}

		//debugf("spaceless literal %q matched with literal %q\n", node.Text, literalValue)
		return types.NewLiteral(literalValue), nil
	})

	visitLiteral := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		return children[0], nil
	})

	// FIXME: this visitor is returning non expression value, which makes us to fallback to any :(
	visitRules := debugHandleExpr(func(node *types.Node, children []any) (any, error) {
		if err := assertNodeToHaveChildrenCount(node, children, 2); err != nil {
			return nil, err
		}

		rules, err := shouldCastAsExpressions(children[1])
		if err != nil {
			return nil, fmt.Errorf("rules: %w", err)
		}

		var knownRuleNames []string
		rulesMap := make(map[string]types.Expression)
		for _, rule := range rules {
			ruleName := rule.ExprName()

			rulesMap[ruleName] = rule
			knownRuleNames = append(knownRuleNames, ruleName)
		}
		for _, rule := range customRules {
			ruleName := rule.ExprName()
			if _, ok := rulesMap[ruleName]; !ok {
				knownRuleNames = append(knownRuleNames, ruleName)
			}
			rulesMap[ruleName] = rule
		}
		for _, k := range knownRuleNames {
			v := rulesMap[k]
			// debugf("resolving refs for %q (known rule names: %q)\n", k, knownRuleNames)
			resolved, err := types.ResolveRefsFor(v, rulesMap)
			if err != nil {
				return nil, fmt.Errorf("resolve refs for %q: %w", k, err)
			}
			rulesMap[k] = resolved
		}

		defaultRule := rulesMap[rules[0].ExprName()]
		rv := types.NewGrammar(rulesMap, defaultRule)
		debugf("loaded %d rules, default rule: %s\n", len(rulesMap), defaultRule)

		return rv, nil
	})

	mux := nodes.NewNodeVisitorMux(nodes.WithDefaultNodeVisitFunc(defaultVisitorWithDebug)).
		HandleExpr("expression", liftChild).
		HandleExpr("term", liftChild).
		HandleExpr("atom", liftChild).
		HandleExpr("parenthesized", visitParenthesized).
		HandleExpr("quantifier", visitQuantifier).
		HandleExpr("quantified", visitQuantified).
		HandleExpr("lookahead_term", visitLookaheadTerm).
		HandleExpr("not_term", visitNotTerm).
		HandleExpr("rule", visitRule).
		HandleExpr("sequence", visitSequence).
		HandleExpr("ored", visitOred).
		HandleExpr("or_term", visitOrTerm).
		HandleExpr("label", visitLabel).
		HandleExpr("reference", visitReference).
		HandleExpr("regex", visitRegex).
		HandleExpr("spaceless_literal", visitSpacelessLiteral).
		HandleExpr("literal", visitLiteral).
		HandleExpr("rules", visitRules)

	return mux
}
