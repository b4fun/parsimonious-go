package nodes

import (
	"fmt"
	"strings"

	"github.com/b4fun/parsimonious-go/types"
)

// NodeVisitFunc is a function that visits a node and its parsed children in the parse tree.
type NodeVisitFunc func(node *types.Node, children []any) (any, error)

// NodeVisitorMux is a multiplexer for visiting nodes in the parse tree.
type NodeVisitorMux struct {
	visitors     map[string]NodeVisitFunc
	defaultVisit NodeVisitFunc
}

// NodeVisitorMuxOpt configures a NodeVisitorMux.
type NodeVisitorMuxOpt func(*NodeVisitorMux)

// WithDefaultNodeVisitFunc sets the default visit function for a NodeVisitorMux.
func WithDefaultNodeVisitFunc(f NodeVisitFunc) NodeVisitorMuxOpt {
	return func(mux *NodeVisitorMux) {
		mux.defaultVisit = f
	}
}

func DefaultNodeVisitor(node *types.Node, children []any) (any, error) {
	if len(children) > 0 {
		return children, nil
	}

	return node, nil
}

// NewNodeVisitorMux creates a NodeVisitorMux instance.
func NewNodeVisitorMux(opts ...NodeVisitorMuxOpt) *NodeVisitorMux {
	rv := &NodeVisitorMux{
		visitors:     make(map[string]NodeVisitFunc),
		defaultVisit: DefaultNodeVisitor,
	}

	for _, opt := range opts {
		opt(rv)
	}

	return rv
}

func (mux *NodeVisitorMux) HandleExpr(
	exprName string,
	f NodeVisitFunc,
) *NodeVisitorMux {
	if _, exists := mux.visitors[exprName]; exists {
		panic(fmt.Sprintf("duplicated visitor for %q", exprName))
	}

	mux.visitors[exprName] = f
	return mux
}

func (mux *NodeVisitorMux) Visit(node *types.Node) (any, error) {
	visitor, ok := mux.visitors[node.Expression.ExprName()]
	if !ok {
		visitor = mux.defaultVisit
	}

	children := make([]any, 0, len(node.Children))
	for _, child := range node.Children {
		c, err := mux.Visit(child)
		if err != nil {
			return nil, err
		}
		children = append(children, c)
	}

	return visitor(node, children)
}

func DumpNodeExprTree(node *types.Node) string {
	sb := new(strings.Builder)

	var dump func(node *types.Node, indent int)

	dump = func(node *types.Node, indent int) {
		sb.WriteString(strings.Repeat(" ", indent))
		fmt.Fprintf(sb, "%s\n", node.Expression)

		for _, child := range node.Children {
			dump(child, indent+2)
		}
	}

	dump(node, 0)

	return sb.String()
}
