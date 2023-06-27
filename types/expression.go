package types

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

func hash(d []byte) uint64 {
	// TODO: use murmur3?
	h := fnv.New64a()
	h.Sum(d)
	return h.Sum64()
}

type nodePosCache map[int]*Node

var nodeInProgress = new(Node)

type nodeCache map[uint64]nodePosCache

func (c nodeCache) get(expr Expression, pos int) *Node {
	if c == nil {
		return nil
	}

	if posCache, ok := c[expr.hash()]; ok {
		if node, ok := posCache[pos]; ok {
			return node
		}
	}

	return nil
}

func (c nodeCache) set(expr Expression, pos int, node *Node) {
	if c == nil {
		return
	}

	exprHash := expr.hash()
	if _, ok := c[exprHash]; !ok {
		c[exprHash] = make(nodePosCache)
	}

	c[exprHash][pos] = node
}

var (
	ErrParseFailed        = fmt.Errorf("parse failed")
	ErrLeftRecursionError = fmt.Errorf("left recursion error") // TODO: add position
)

type matchResult struct {
	Node *Node
	Err  error
}

func (mr *matchResult) isNoMatch() bool {
	return mr.Node == nil && mr.Err == nil
}

func (mr *matchResult) isMatchedNode() bool {
	return mr.Node != nil && mr.Err == nil
}

func (mr *matchResult) isMatchFailed() bool {
	return mr.Node == nil && mr.Err != nil
}

func (mr *matchResult) String() string {
	if mr.Err != nil {
		return fmt.Sprintf("matchResult{Err: %s}", mr.Err)
	}
	if mr.Node != nil {
		return fmt.Sprintf(
			"matchResult{NodeMatchedStart: %d, NodeMatchedEnd: %d, NodeExpr: %q}",
			mr.Node.Start, mr.Node.End, string(mr.Node.Expression.ExprName()),
		)
	}
	return "matchResult{NoMatch}"
}

func noMatch() *matchResult {
	return &matchResult{
		Node: nil,
		Err:  nil,
	}
}

func matchedNode(node *Node) *matchResult {
	return &matchResult{
		Node: node,
		Err:  nil,
	}
}

func matchFailed(err error) *matchResult {
	return &matchResult{
		Node: nil,
		Err:  err,
	}
}

func formatRuleRHSWithOptionalName(name string, rhs string) string {
	if name == "" {
		return rhs
	}
	return fmt.Sprintf("%s = %s", name, rhs)
}

func joinExpressionAsRule(expr Expression) string {
	exprRepr := expr.ExprName()
	if exprRepr == "" {
		return expr.String()
	}
	return exprRepr
}

func joinExpressionsAsRule(exprs []Expression, sep string) string {
	var sb strings.Builder
	for i, expr := range exprs {
		if i > 0 {
			sb.WriteString(sep)
		}
		exprRepr := expr.ExprName()
		if exprRepr == "" {
			sb.WriteString(expr.String())
		} else {
			sb.WriteString(exprRepr)
		}
	}
	return sb.String()
}

// Expression represents a parsimonious expression.
type Expression interface {
	fmt.Stringer

	// ExprName returns the name of the expression.
	ExprName() string
	// SetExprName sets the name of the expression.
	// TODO: maybe we should get rid of this?
	SetExprName(string)
	// Match matches the expression against the given text at the given rune position.
	Match(text string, parseOpts *ParseOptions) (*Node, error)

	// matchWithCache matches the expression against the given text at the given rune position. (internal usage)
	matchWithCache(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult
	// hash returns a hash value for the expression. (internal usage)
	hash() uint64
}

// ParseOptions represents options for parsing.
type ParseOptions struct {
	pos   int
	debug bool
}

func (opts *ParseOptions) withPos(newPos int) *ParseOptions {
	return &ParseOptions{
		pos:   newPos,
		debug: opts.debug,
	}
}

func (opts *ParseOptions) debugf(format string, args ...interface{}) { //nolint:unused
	if opts.debug {
		fmt.Printf(format, args...)
	}
}

func createParseOpts(opts ...ParseOption) *ParseOptions {
	parseOpts := &ParseOptions{}
	for _, o := range opts {
		o(parseOpts)
	}
	return parseOpts
}

// ParseOption configures a ParseOptions.
type ParseOption func(*ParseOptions)

// ParseWithDebug enables debug mode on parsing.
func ParseWithDebug(debug bool) func(*ParseOptions) {
	return func(opts *ParseOptions) {
		opts.debug = debug
	}
}

// ParseWithExpression parses the given text with the given expression.
func ParseWithExpression(expr Expression, text string, opts ...ParseOption) (*Node, error) {
	parseOpts := createParseOpts(opts...)

	node, err := expr.Match(text, parseOpts)
	if err != nil {
		return nil, err
	}
	if textLen := utf8.RuneCountInString(text); node.End < textLen {
		return nil, fmt.Errorf(
			"incomplete input parsed, parsed end=%d, input length=%d",
			node.End, textLen,
		)
	}

	return node, nil
}

type withResolveRefs interface {
	Expression

	ResolveRefs(rules map[string]Expression) (Expression, error)
}

func ResolveRefsFor(v Expression, rules map[string]Expression) (Expression, error) {
	expr, ok := v.(withResolveRefs)
	if !ok {
		return v, nil
	}

	return expr.ResolveRefs(rules)
}

func resolveRefsForMany(vs []Expression, rules map[string]Expression) ([]Expression, error) {
	var resolved []Expression
	for _, v := range vs {
		expr, err := ResolveRefsFor(v, rules)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, expr)
	}

	return resolved, nil
}

type exprImpl interface {
	exprName() string
	setExprName(s string)
	identity() []byte // for hashing
	uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult
	asRule() string
}

type expression struct {
	impl exprImpl
}

func (e *expression) ExprName() string {
	return e.impl.exprName()
}

func (e *expression) SetExprName(n string) {
	e.impl.setExprName(n)
}

func (e *expression) Match(text string, parseOpts *ParseOptions) (*Node, error) {
	cache := new(nodeCache)
	result := e.matchWithCache(text, parseOpts, *cache)
	switch {
	case result.isMatchedNode():
		return result.Node, nil
	case result.isMatchFailed():
		return nil, result.Err
	default:
		return nil, fmt.Errorf("%w: text=%s, pos=%d", ErrParseFailed, text, parseOpts.pos)
	}
}

func (e *expression) matchWithCache(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	node := cache.get(e, parseOpts.pos)
	if node == nil {
		cache.set(e, parseOpts.pos, nodeInProgress)
		matchResult := e.impl.uncachedMatch(text, parseOpts, cache)
		if matchResult.isMatchFailed() {
			return matchResult
		}
		node = matchResult.Node
		cache.set(e, parseOpts.pos, node)
	}
	if node == nodeInProgress {
		return matchFailed(fmt.Errorf("%w: text=%s, pos=%d", ErrLeftRecursionError, text, parseOpts.pos))
	}
	if node == nil {
		return noMatch()
	}

	return matchedNode(node)
}

func (e *expression) hash() uint64 {
	return hash(e.impl.identity())
}

func (e *expression) String() string {
	return fmt.Sprintf(
		"<%T %s>",
		e.impl,
		e.impl.asRule(),
	)
}

type Literal struct {
	expression

	literal          string
	literalRuneCount int
	name             string
}

var _ Expression = (*Literal)(nil)
var _ exprImpl = (*Literal)(nil)

func NewLiteralWithName(name string, literal string) *Literal {
	rv := &Literal{
		literal:          literal,
		literalRuneCount: utf8.RuneCountInString(literal),
		name:             name,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func NewLiteral(literal string) *Literal {
	return NewLiteralWithName("", literal)
}

func (l *Literal) GetLiteral() string {
	return l.literal
}

func (l *Literal) exprName() string {
	return l.name
}

func (l *Literal) setExprName(n string) {
	l.name = n
}

func (l *Literal) identity() []byte {
	return []byte("literal:" + l.name + ":" + l.literal)
}

func (l *Literal) uncachedMatch(text string, parseOpts *ParseOptions, _ nodeCache) *matchResult {
	pos := parseOpts.pos
	if utf8.RuneCountInString(text) < pos+l.literalRuneCount {
		return noMatch()
	}

	if sliceStringAsRuneSlice(text, pos, pos+l.literalRuneCount) == l.literal {
		node := newNode(l, text, pos, pos+l.literalRuneCount)
		return matchedNode(node)
	}

	return noMatch()
}

func (l *Literal) asRule() string {
	return formatRuleRHSWithOptionalName(
		l.name,
		fmt.Sprintf("%q", l.literal),
	)
}

type Sequence struct {
	expression

	name    string
	members []Expression
}

var _ Expression = (*Sequence)(nil)
var _ exprImpl = (*Sequence)(nil)
var _ withResolveRefs = (*Sequence)(nil)

func NewSequence(name string, members []Expression) *Sequence {
	rv := &Sequence{
		name:    name,
		members: members,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func (s *Sequence) exprName() string {
	return s.name
}

func (s *Sequence) setExprName(n string) {
	s.name = n
}

func (s *Sequence) identity() []byte {
	return []byte("sequence:" + s.name)
}

func (s *Sequence) uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	curPos := parseOpts.pos
	children := make([]*Node, 0, len(s.members))
	for idx := range s.members {
		matchResult := s.members[idx].matchWithCache(text, parseOpts.withPos(curPos), cache)
		if matchResult.isMatchFailed() {
			return matchResult
		}
		if matchResult.isNoMatch() {
			return matchResult
		}
		node := matchResult.Node
		children = append(children, node)
		curPos += node.End - node.Start
	}

	node := newNodeWithChildren(s, text, parseOpts.pos, curPos, children)
	return matchedNode(node)
}

func (s *Sequence) ResolveRefs(refs map[string]Expression) (Expression, error) {
	newMembers, err := resolveRefsForMany(s.members, refs)
	if err != nil {
		return nil, err
	}

	s.members = newMembers
	return s, nil
}

func (s *Sequence) asRule() string {
	return formatRuleRHSWithOptionalName(
		s.exprName(),
		fmt.Sprintf(
			"(%s)",
			joinExpressionsAsRule(s.members, " "),
		),
	)
}

type OneOf struct {
	expression

	name    string
	members []Expression
}

var _ Expression = (*OneOf)(nil)
var _ exprImpl = (*OneOf)(nil)
var _ withResolveRefs = (*OneOf)(nil)

func NewOneOf(name string, members []Expression) *OneOf {
	rv := &OneOf{
		name:    name,
		members: members,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func (of *OneOf) SetMembers(members []Expression) {
	of.members = members
}

func (of *OneOf) exprName() string {
	return of.name
}

func (of *OneOf) setExprName(n string) {
	of.name = n
}

func (of *OneOf) identity() []byte {
	return []byte("oneOf:" + of.name)
}

func (of *OneOf) uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	for idx := range of.members {
		matchResult := of.members[idx].matchWithCache(text, parseOpts, cache)
		if matchResult.isMatchFailed() {
			return matchResult
		}
		if matchResult.isMatchedNode() {
			oneOfNode := newNodeWithChildren(of, text, parseOpts.pos, matchResult.Node.End, []*Node{matchResult.Node})
			return matchedNode(oneOfNode)
		}
	}

	return noMatch()
}

func (of *OneOf) ResolveRefs(refs map[string]Expression) (Expression, error) {
	newMembers, err := resolveRefsForMany(of.members, refs)
	if err != nil {
		return nil, err
	}

	of.members = newMembers
	return of, nil
}

func (of *OneOf) asRule() string {
	return formatRuleRHSWithOptionalName(
		of.exprName(),
		fmt.Sprintf(
			"(%s)",
			joinExpressionsAsRule(of.members, " / "),
		),
	)
}

type Lookahead struct {
	expression

	name     string
	member   Expression
	negative bool
}

var _ Expression = (*Lookahead)(nil)
var _ exprImpl = (*Lookahead)(nil)
var _ withResolveRefs = (*Lookahead)(nil)

func NewLookahead(name string, member Expression, negative bool) *Lookahead {
	rv := &Lookahead{
		name:     name,
		member:   member,
		negative: negative,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func NewNot(member Expression) *Lookahead {
	return NewLookahead("", member, true)
}

func (l *Lookahead) exprName() string {
	return l.name
}

func (l *Lookahead) setExprName(n string) {
	l.name = n
}

func (l *Lookahead) identity() []byte {
	return []byte("lookahead:" + l.name)
}

func (l *Lookahead) uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	matchResult := l.member.matchWithCache(text, parseOpts, cache)
	if matchResult.isMatchFailed() {
		return matchResult
	}

	pos := parseOpts.pos
	switch {
	case matchResult.isNoMatch() && l.negative:
		return matchedNode(newNode(l, text, pos, pos))
	case matchResult.isMatchedNode() && !l.negative:
		return matchedNode(newNode(l, text, pos, pos))
	default:
		return noMatch()
	}
}

func (l *Lookahead) ResolveRefs(refs map[string]Expression) (Expression, error) {
	newMember, err := ResolveRefsFor(l.member, refs)
	if err != nil {
		return nil, err
	}

	l.member = newMember
	return l, nil
}

func (l *Lookahead) asRule() string {
	prefix := "&"
	if l.negative {
		prefix = "!"
	}

	return formatRuleRHSWithOptionalName(
		l.exprName(),
		fmt.Sprintf(
			"(%s%s)",
			prefix,
			joinExpressionAsRule(l.member),
		),
	)
}

type Quantifier struct {
	expression

	name   string
	member Expression
	min    float64
	max    float64
}

var _ Expression = (*Quantifier)(nil)
var _ exprImpl = (*Quantifier)(nil)
var _ withResolveRefs = (*Quantifier)(nil)

func newQuantifier(name string, member Expression, min float64, max float64) *Quantifier {
	rv := &Quantifier{
		name:   name,
		member: member,
		min:    min,
		max:    max,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func NewZeroOrMore(name string, member Expression) *Quantifier {
	return newQuantifier(name, member, 0, math.Inf(1))
}

func NewOneOrMore(name string, member Expression) *Quantifier {
	return newQuantifier(name, member, 1, math.Inf(1))
}

func NewOptional(name string, member Expression) *Quantifier {
	return newQuantifier(name, member, 0, 1)
}

func (q *Quantifier) exprName() string {
	return q.name
}

func (q *Quantifier) setExprName(n string) {
	q.name = n
}

func (q *Quantifier) identity() []byte {
	return []byte("quantifier:" + q.name)
}

func (q *Quantifier) uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	curPos := parseOpts.pos
	children := make([]*Node, 0)
	size := utf8.RuneCountInString(text)
	for curPos < size && float64(len(children)) < q.max {
		matchResult := q.member.matchWithCache(text, parseOpts.withPos(curPos), cache)
		if matchResult.isMatchFailed() {
			return matchResult
		}
		if matchResult.isNoMatch() {
			break
		}
		node := matchResult.Node
		//parseOpts.debugf("[%s] matched new node: %s %q\n", q, node, node.Text)
		children = append(children, node)
		nodeMatchedLength := node.End - node.Start
		if nodeMatchedLength == 0 && float64(len(children)) >= q.min {
			// This is a zero-length match (lookahead), so we need to advance the cursor after reaching minimum
			break
		}
		curPos += nodeMatchedLength
	}

	if float64(len(children)) < q.min {
		return noMatch()
	}

	node := newNodeWithChildren(q, text, parseOpts.pos, curPos, children)
	return matchedNode(node)
}

func (q *Quantifier) ResolveRefs(refs map[string]Expression) (Expression, error) {
	newMember, err := ResolveRefsFor(q.member, refs)
	if err != nil {
		return nil, err
	}

	q.member = newMember
	return q, nil
}

func (q *Quantifier) asRule() string {
	var quantifier string
	switch {
	case q.min == 0 && q.max == 1:
		quantifier = "?"
	case q.min == 0 && q.max == math.Inf(1):
		quantifier = "*"
	case q.min == 1 && q.max == math.Inf(1):
		quantifier = "+"
	case q.max == math.Inf(1):
		quantifier = fmt.Sprintf("{%d,}", int(q.min))
	case q.min == 0:
		quantifier = fmt.Sprintf("{,%d}", int(q.max))
	default:
		quantifier = fmt.Sprintf("{%d,%d}", int(q.min), int(q.max))
	}

	return formatRuleRHSWithOptionalName(
		q.exprName(),
		fmt.Sprintf("%s%s", q.member, quantifier),
	)
}

type Regex struct {
	expression

	name string
	re   *regexp2.Regexp
}

func NewRegex(name string, re *regexp2.Regexp) *Regex {
	rv := &Regex{
		name: name,
		re:   re,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func (r *Regex) exprName() string {
	return r.name
}

func (r *Regex) setExprName(n string) {
	r.name = n
}

func (r *Regex) identity() []byte {
	return []byte("re:" + r.name + ":" + r.re.String())
}

func (r *Regex) uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	pos := parseOpts.pos
	textToMatch := sliceStringAsRuneSlice(text, pos, -1)

	//parseOpts.debugf("[%s] trying regex (%s) match at pos %d %q\n",r, r.re,pos, textToMatch)

	matchGroups, err := r.re.FindStringMatch(textToMatch)
	if err != nil {
		//parseOpts.debugf("[%s] regex match failed: %s (pos=%d)\n", r, err, pos)

		return matchFailed(err)
	}
	if matchGroups == nil {
		//parseOpts.debugf("[%s] regex match failed: no match (pos=%d)\n", r, pos)

		return noMatch()
	}
	if len(matchGroups.Captures) < 1 {
		//parseOpts.debugf("[%s] regex match failed: no match (pos=%d)\n", r, pos)

		return noMatch()
	}

	match := matchGroups.Captures[0]
	matchedEnd := pos + match.Index + match.Length

	//parseOpts.debugf("[%s] regex matched: (pos=%d)\n", r, pos)
	node := newRegexNode(r, text, pos, matchedEnd, match.String())
	return matchedNode(node)
}

func (r *Regex) asRule() string {
	// TODO: record options
	return formatRuleRHSWithOptionalName(
		r.exprName(),
		fmt.Sprintf("~%s", r.re.String()),
	)
}

type LazyReference struct {
	expression

	name          string
	referenceName string
}

var _ Expression = (*LazyReference)(nil)
var _ exprImpl = (*LazyReference)(nil)
var _ withResolveRefs = (*LazyReference)(nil)

func NewLazyReference(referenceName string) *LazyReference {
	rv := &LazyReference{
		name:          "lazy_reference",
		referenceName: referenceName,
	}
	rv.expression = expression{impl: rv}

	return rv
}

func (r *LazyReference) exprName() string {
	return r.name
}

func (r *LazyReference) setExprName(n string) {
	r.name = n
}

func (r *LazyReference) identity() []byte {
	return []byte("lazy_reference:" + r.referenceName)
}

func (r *LazyReference) uncachedMatch(text string, parseOpts *ParseOptions, cache nodeCache) *matchResult {
	return matchFailed(fmt.Errorf("lazy reference %q is not resolved", r.referenceName))
}

func (r *LazyReference) ResolveRefs(refs map[string]Expression) (Expression, error) {
	seenExprs := make(map[string]struct{})
	current := r
	for {
		currentIdentity := string(current.identity())
		if _, exists := seenExprs[currentIdentity]; exists {
			return nil, fmt.Errorf("circular reference detected for %q", r.referenceName)
		} else {
			seenExprs[currentIdentity] = struct{}{}
		}
		resolved, exists := refs[current.referenceName]
		if !exists {
			return nil, fmt.Errorf("lazy reference %q is not resolved", r.referenceName)
		}
		if resolvedReference, ok := resolved.(*LazyReference); ok {
			current = resolvedReference
			continue
		}
		return resolved, nil
	}
}

func (r *LazyReference) asRule() string {
	return fmt.Sprintf("<LazyReference to %s>", r.referenceName)
}
