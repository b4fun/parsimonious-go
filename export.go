package parsimonious

import (
	"github.com/b4fun/parsimonious-go/internal/bootstrap"
	"github.com/b4fun/parsimonious-go/nodes"
	"github.com/b4fun/parsimonious-go/types"
)

var (
	NewGrammar          = bootstrap.NewGrammar
	ParsimoniousGrammar = bootstrap.ParsimoniousGrammar

	ParseWithDebug = types.ParseWithDebug

	DumpNodeExprTree         = nodes.DumpNodeExprTree
	NewNodeVisitorMux        = nodes.NewNodeVisitorMux
	WithDefaultNodeVisitFunc = nodes.WithDefaultNodeVisitFunc
)

type (
	Node       = types.Node
	Expression = types.Expression

	ErrParseFailed		   = types.ErrParseFailed
	ErrIncompleteParseFailed = types.ErrIncompleteParseFailed
)
