package types

import (
	"fmt"
)

// Node represents a node in the parse tree.
type Node struct {
	// Expression is the expression that matched this node.
	Expression Expression
	// Text is the text that matched this node.
	Text string
	// Start is the rune start index of the match.
	Start int
	// End is the rune end index of the match.
	End int
	// Children are the child nodes of this node.
	Children []*Node
	// Match is the string that matched this node from the regex expression.
	Match string
}

func (n *Node) String() string {
	return fmt.Sprintf(
		"<Node: %s start:%d, end:%d children:%d>\n",
		n.Expression, n.Start, n.End, len(n.Children),
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
		Text:       sliceStringAsRuneSlice(fullText, start, end),
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
