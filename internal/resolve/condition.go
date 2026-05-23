package resolve

import (
	"fmt"
	"strconv"
	"strings"
)

// Expr is the AST for a condition expression. Produced by ParseCondition,
// consumed by EvaluateCondition. The AST is not serialized — the descriptor
// stores the source string and we reparse on first use per CLI invocation.
type Expr interface {
	exprNode()
}

type LiteralExpr struct {
	Value any // bool, float64, string, or nil
}

// ReferenceExpr is a path into the bindings map. The path syntax matches
// what params: references already support (e.g. "step", "step.field",
// "step[0].field"). Resolution is delegated to ResolveReference.
type ReferenceExpr struct {
	Path string
}

type NotExpr struct {
	Operand Expr
}

type BinaryExpr struct {
	Op    string // "==", "!=", "<", "<=", ">", ">=", "&&", "||"
	Left  Expr
	Right Expr
}

func (LiteralExpr) exprNode()   {}
func (ReferenceExpr) exprNode() {}
func (NotExpr) exprNode()       {}
func (BinaryExpr) exprNode()    {}

// ParseCondition parses a condition expression. Grammar (lowest to highest
// precedence): or → and → not → comparison → primary, where primary is
// (expr) | literal | reference. Strings must be quoted; bare words are
// references; true/false/null are literals.
func ParseCondition(src string) (Expr, error) {
	p := &condParser{src: src}
	p.skipWhitespace()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("empty condition")
	}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos < len(p.src) {
		return nil, fmt.Errorf("unexpected token at position %d: %q", p.pos, p.src[p.pos:])
	}
	return expr, nil
}

type condParser struct {
	src string
	pos int
}

func (p *condParser) skipWhitespace() {
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
			continue
		}
		break
	}
}

func (p *condParser) consume(s string) bool {
	p.skipWhitespace()
	if strings.HasPrefix(p.src[p.pos:], s) {
		p.pos += len(s)
		return true
	}
	return false
}

func (p *condParser) parseOr() (Expr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.consume("||") {
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: "||", Left: left, Right: right}
	}
	return left, nil
}

func (p *condParser) parseAnd() (Expr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.consume("&&") {
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = BinaryExpr{Op: "&&", Left: left, Right: right}
	}
	return left, nil
}

func (p *condParser) parseNot() (Expr, error) {
	p.skipWhitespace()
	// `!=` is a comparison, not a `!`. Don't eat `!` here if `=` follows.
	if p.pos < len(p.src) && p.src[p.pos] == '!' && (p.pos+1 >= len(p.src) || p.src[p.pos+1] != '=') {
		p.pos++
		operand, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return NotExpr{Operand: operand}, nil
	}
	return p.parseComparison()
}

func (p *condParser) parseComparison() (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	// Longest match first; "<=" before "<", etc.
	for _, op := range []string{"==", "!=", "<=", ">=", "<", ">"} {
		if p.consume(op) {
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return BinaryExpr{Op: op, Left: left, Right: right}, nil
		}
	}
	return left, nil
}

func (p *condParser) parsePrimary() (Expr, error) {
	p.skipWhitespace()
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unexpected end of expression")
	}
	c := p.src[p.pos]
	switch {
	case c == '(':
		p.pos++
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if !p.consume(")") {
			return nil, fmt.Errorf("expected ')' at position %d", p.pos)
		}
		return expr, nil
	case c == '"':
		return p.parseStringLiteral()
	case c == '-' || (c >= '0' && c <= '9'):
		return p.parseNumberLiteral()
	case isIdentStart(c):
		return p.parseIdentOrReference()
	}
	return nil, fmt.Errorf("unexpected character %q at position %d", c, p.pos)
}

func (p *condParser) parseStringLiteral() (Expr, error) {
	if p.src[p.pos] != '"' {
		return nil, fmt.Errorf("expected '\"' at position %d", p.pos)
	}
	p.pos++
	var buf strings.Builder
	for p.pos < len(p.src) && p.src[p.pos] != '"' {
		if p.src[p.pos] == '\\' && p.pos+1 < len(p.src) {
			switch p.src[p.pos+1] {
			case 'n':
				buf.WriteByte('\n')
			case 't':
				buf.WriteByte('\t')
			case '"':
				buf.WriteByte('"')
			case '\\':
				buf.WriteByte('\\')
			default:
				buf.WriteByte(p.src[p.pos+1])
			}
			p.pos += 2
			continue
		}
		buf.WriteByte(p.src[p.pos])
		p.pos++
	}
	if p.pos >= len(p.src) {
		return nil, fmt.Errorf("unterminated string literal")
	}
	p.pos++ // closing "
	return LiteralExpr{Value: buf.String()}, nil
}

func (p *condParser) parseNumberLiteral() (Expr, error) {
	start := p.pos
	if p.src[p.pos] == '-' {
		p.pos++
	}
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if (c >= '0' && c <= '9') || c == '.' {
			p.pos++
			continue
		}
		break
	}
	numStr := p.src[start:p.pos]
	n, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q at position %d", numStr, start)
	}
	return LiteralExpr{Value: n}, nil
}

func (p *condParser) parseIdentOrReference() (Expr, error) {
	start := p.pos
	for p.pos < len(p.src) && isIdentPart(p.src[p.pos]) {
		p.pos++
	}
	// Then accept any number of .ident or [n] accessors, mirroring the
	// path syntax recognized by ResolveReference.
	for p.pos < len(p.src) {
		c := p.src[p.pos]
		if c == '.' {
			p.pos++
			for p.pos < len(p.src) && isIdentPart(p.src[p.pos]) {
				p.pos++
			}
		} else if c == '[' {
			p.pos++
			for p.pos < len(p.src) && p.src[p.pos] != ']' {
				p.pos++
			}
			if p.pos >= len(p.src) {
				return nil, fmt.Errorf("unterminated '[' in reference")
			}
			p.pos++ // ]
		} else {
			break
		}
	}
	tok := p.src[start:p.pos]
	switch tok {
	case "true":
		return LiteralExpr{Value: true}, nil
	case "false":
		return LiteralExpr{Value: false}, nil
	case "null":
		return LiteralExpr{Value: nil}, nil
	}
	return ReferenceExpr{Path: tok}, nil
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// EvaluateCondition resolves expr against bindings and asserts the final
// value is a bool. A bare-reference condition only passes if the bound
// value is itself a bool — no JS-style truthy coercion.
func EvaluateCondition(expr Expr, bindings map[string]any) (bool, error) {
	val, err := evalExpr(expr, bindings)
	if err != nil {
		return false, err
	}
	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("condition must evaluate to a bool, got %T (%v)", val, val)
	}
	return b, nil
}

// EvaluateExpression resolves expr against bindings and returns the raw
// value without asserting bool. Use this for let-step value bindings
// (where any type is fine); use EvaluateCondition for if-step conditions.
func EvaluateExpression(expr Expr, bindings map[string]any) (any, error) {
	return evalExpr(expr, bindings)
}

func evalExpr(expr Expr, bindings map[string]any) (any, error) {
	switch e := expr.(type) {
	case LiteralExpr:
		return e.Value, nil
	case ReferenceExpr:
		val, isRef, err := ResolveReference(e.Path, bindings)
		if err != nil {
			return nil, err
		}
		if !isRef {
			return nil, fmt.Errorf("reference %q not in scope", e.Path)
		}
		return val, nil
	case NotExpr:
		v, err := evalExpr(e.Operand, bindings)
		if err != nil {
			return nil, err
		}
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("'!' requires a bool operand, got %T (%v)", v, v)
		}
		return !b, nil
	case BinaryExpr:
		return evalBinary(e, bindings)
	}
	return nil, fmt.Errorf("unknown expression type %T", expr)
}

func evalBinary(e BinaryExpr, bindings map[string]any) (any, error) {
	if e.Op == "&&" || e.Op == "||" {
		l, err := evalExpr(e.Left, bindings)
		if err != nil {
			return nil, err
		}
		lb, ok := l.(bool)
		if !ok {
			return nil, fmt.Errorf("'%s' left operand must be bool, got %T (%v)", e.Op, l, l)
		}
		if e.Op == "&&" && !lb {
			return false, nil
		}
		if e.Op == "||" && lb {
			return true, nil
		}
		r, err := evalExpr(e.Right, bindings)
		if err != nil {
			return nil, err
		}
		rb, ok := r.(bool)
		if !ok {
			return nil, fmt.Errorf("'%s' right operand must be bool, got %T (%v)", e.Op, r, r)
		}
		return rb, nil
	}

	l, err := evalExpr(e.Left, bindings)
	if err != nil {
		return nil, err
	}
	r, err := evalExpr(e.Right, bindings)
	if err != nil {
		return nil, err
	}

	switch e.Op {
	case "==":
		return condEquals(l, r), nil
	case "!=":
		return !condEquals(l, r), nil
	case "<", "<=", ">", ">=":
		ln, rn, err := condNumbers(l, r)
		if err != nil {
			return nil, fmt.Errorf("'%s' requires numeric operands: %v", e.Op, err)
		}
		switch e.Op {
		case "<":
			return ln < rn, nil
		case "<=":
			return ln <= rn, nil
		case ">":
			return ln > rn, nil
		case ">=":
			return ln >= rn, nil
		}
	}
	return nil, fmt.Errorf("unknown operator %q", e.Op)
}

// condEquals compares primitives by value. Numbers are compared
// numerically regardless of int/float repr (JSON roundtrip lands on
// float64; YAML may produce int64/uint64).
func condEquals(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	an, aok := condNumber(a)
	bn, bok := condNumber(b)
	if aok && bok {
		return an == bn
	}
	return a == b
}

func condNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}

func condNumbers(a, b any) (float64, float64, error) {
	an, aok := condNumber(a)
	if !aok {
		return 0, 0, fmt.Errorf("left is %T", a)
	}
	bn, bok := condNumber(b)
	if !bok {
		return 0, 0, fmt.Errorf("right is %T", b)
	}
	return an, bn, nil
}

// conditionRootRefs walks expr and collects the root id of every reference
// (the part before any `.` or `[`). Used by the reconciler to statically
// validate that an if-step's condition only depends on ids in scope.
func conditionRootRefs(expr Expr) []string {
	var out []string
	var walk func(Expr)
	walk = func(e Expr) {
		switch n := e.(type) {
		case ReferenceExpr:
			out = append(out, referenceRoot(n.Path))
		case NotExpr:
			walk(n.Operand)
		case BinaryExpr:
			walk(n.Left)
			walk(n.Right)
		}
	}
	walk(expr)
	return out
}

func referenceRoot(path string) string {
	for i := 0; i < len(path); i++ {
		if path[i] == '.' || path[i] == '[' {
			return path[:i]
		}
	}
	return path
}
