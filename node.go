package parsimonious

import (
	"fmt"
	"strings"
)

type Node struct {
	Expression Expression
	Text       string
	Start      int
	End        int
	Children   []*Node
	Match      string // for regex expression
}

func (n *Node) String() string {
	return fmt.Sprintf(
		"<Node: %s start:%d, end:%d>\n",
		n.Expression, n.Start, n.End,
	)
}

func newNode(
	expression Expression,
	fullText string,
	start int,
	end int,
) *Node {
	return &Node{
		Expression: expression,
		Text:       fullText[start:end],
		Start:      start,
		End:        end,
		Children:   make([]*Node, 0),
	}
}

func newNodeWithChildren(
	expression Expression,
	fullText string,
	start int,
	end int,
	children []*Node,
) *Node {
	node := newNode(expression, fullText, start, end)
	node.Children = children
	return node
}

func newRegexNode(
	expression Expression,
	fullText string,
	start int,
	end int,
	match string,
) *Node {
	node := newNode(expression, fullText, start, end)
	node.Match = match
	return node
}

type NodeVisitor interface {
	Visit(node *Node) (any, error)
}

type NodeVisitFunc func(v NodeVisitor, node *Node) (any, error)

type NodeVisitWithChildrenFunc func(node *Node, children []any) (any, error)

type NodeVisitorMux struct {
	visitors     map[string]NodeVisitFunc
	defaultVisit NodeVisitFunc
}

func NewNodeVisitorMux(defaultVisit NodeVisitFunc) *NodeVisitorMux {
	return &NodeVisitorMux{
		visitors:     make(map[string]NodeVisitFunc),
		defaultVisit: defaultVisit,
	}
}

func (mux *NodeVisitorMux) VisitWith(
	exprName string,
	f NodeVisitFunc,
) *NodeVisitorMux {
	if _, exists := mux.visitors[exprName]; exists {
		panic(fmt.Sprintf("duplicated visitor for %q", exprName))
	}

	mux.visitors[exprName] = f
	return mux
}

func visitWithChildren(f NodeVisitWithChildrenFunc) NodeVisitFunc {
	return func(v NodeVisitor, node *Node) (any, error) {
		children := make([]any, 0, len(node.Children))
		for _, child := range node.Children {
			c, err := v.Visit(child)
			if err != nil {
				return nil, err
			}
			children = append(children, c)
		}

		return f(node, children)
	}
}

func (mux *NodeVisitorMux) VisitWithChildren(
	exprName string,
	f NodeVisitWithChildrenFunc,
) *NodeVisitorMux {
	return mux.VisitWith(exprName, visitWithChildren(f))
}

func (mux *NodeVisitorMux) Visit(
	node *Node,
) (any, error) {
	visitor, ok := mux.visitors[node.Expression.ExprName()]
	if !ok {
		visitor = mux.defaultVisit
	}

	return visitor(mux, node)
}

func assertNodeToHaveChildrenCount(node *Node, children []any, count int) error {
	if len(node.Children) == count {
		return nil
	}

	return fmt.Errorf(
		"%s should have %d children, got %d",
		node, count, len(children),
	)
}

func DumpNodeExprTree(node *Node) string {
	sb := new(strings.Builder)

	var dump func(node *Node, indent int)

	dump = func(node *Node, indent int) {
		sb.WriteString(strings.Repeat(" ", indent))
		fmt.Fprintf(sb, "%s\n", node.Expression)

		for _, child := range node.Children {
			dump(child, indent+2)
		}
	}

	dump(node, 0)

	return sb.String()
}
