package bootstrap

import (
	"fmt"

	"github.com/b4fun/parsimonious-go/types"
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

var spacelessLiteral = types.NewOneOf(
	"spaceless_literal",
	[]types.Expression{
		types.NewRegex(
			"",
			regexp2.MustCompile(`(?si)^r?"[^"\\]*(?:\\.[^"\\]*)*"`, regexp2.RE2),
		),
		types.NewRegex(
			"",
			regexp2.MustCompile(`(?si)^r?'[^'\\]*(?:\\.[^'\\]*)*'`, regexp2.RE2),
		),
	},
)

var ParsimoniousGrammar *types.Grammar

func initParsimoniousGrammar() (*types.Grammar, error) {
	const debug = false

	mux := createRuleVisitor(debug, []types.Expression{spacelessLiteral})
	bootstrapTree, err := types.ParseWithExpression(
		createBootstrapRules(),
		ruleSyntax,
		types.ParseWithDebug(debug),
	)
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
	ParsimoniousGrammar, err = initParsimoniousGrammar()
	if err != nil {
		panic(fmt.Errorf("init parsimonious grammar: %w", err))
	}
}

func NewGrammar(input string, parseOpts ...types.ParseOption) (*types.Grammar, error) {
	tree, err := ParsimoniousGrammar.Parse(input, parseOpts...)
	if err != nil {
		return nil, fmt.Errorf("parse grammar: %w", err)
	}

	const debugRuleVisitor = false
	mux := createRuleVisitor(debugRuleVisitor, nil)
	return asGrammar(mux.Visit(tree))
}
