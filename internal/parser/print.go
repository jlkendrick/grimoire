package parser

import (
	"fmt"
	"strings"
)

// String renders the parsed AST as nested struct literals (field names and values).
func (r *Ritual) String() string {
	if r == nil {
		return "nil"
	}
	var p astPrinter
	p.printRitual(r)
	return p.s.String()
}

type astPrinter struct {
	s     strings.Builder
	depth int
}

func (p *astPrinter) pad() string {
	return strings.Repeat("\t", p.depth)
}

func (p *astPrinter) line(text string) {
	p.s.WriteString(p.pad())
	p.s.WriteString(text)
	p.s.WriteByte('\n')
}

func (p *astPrinter) printRitual(r *Ritual) {
	p.line("Ritual{")
	p.depth++
	p.line(fmt.Sprintf("Name: %q,", r.Name))
	p.printStmtSlice("Body", r.Body)
	p.depth--
	p.line("}")
}

func (p *astPrinter) printStmtSlice(name string, stmts []*Stmt) {
	if len(stmts) == 0 {
		p.line(name + ": []*Stmt{},")
		return
	}
	p.line(name + ": []*Stmt{")
	p.depth++
	for _, stmt := range stmts {
		p.printStmt(stmt)
	}
	p.depth--
	p.line("},")
}

func (p *astPrinter) printStmt(stmt *Stmt) {
	if stmt == nil {
		p.line("nil,")
		return
	}
	p.line("&Stmt{")
	p.depth++
	switch {
	case stmt.Let != nil:
		p.printLet(stmt.Let)
	case stmt.If != nil:
		p.printIf(stmt.If)
	case stmt.Call != nil:
		p.printCall(stmt.Call)
	}
	p.depth--
	p.line("},")
}

func (p *astPrinter) printLet(l *LetStmt) {
	p.line("Let: &LetStmt{")
	p.depth++
	p.line(fmt.Sprintf("Ident: %q,", l.Ident))
	p.printExprField("Expr", l.Expr)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printIf(i *IfStmt) {
	p.line("If: &IfStmt{")
	p.depth++
	p.printExprField("Condition", i.Condition)
	p.printStmtSlice("TrueBlock", i.TrueBlock)
	p.printStmtSlice("ElseBlock", i.ElseBlock)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printCall(c *CallExpr) {
	p.line("Call: &CallExpr{")
	p.depth++
	p.line(fmt.Sprintf("Name: %q,", c.Name))
	p.printArgSlice("Args", c.Args)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printArgSlice(name string, args []*Argument) {
	if len(args) == 0 {
		p.line(name + ": []*Argument{},")
		return
	}
	p.line(name + ": []*Argument{")
	p.depth++
	for _, arg := range args {
		p.printArgument(arg)
	}
	p.depth--
	p.line("},")
}

func (p *astPrinter) printArgument(a *Argument) {
	if a == nil {
		p.line("nil,")
		return
	}
	p.line("&Argument{")
	p.depth++
	p.line(fmt.Sprintf("Name: %q,", a.Name))
	p.printExprField("Value", a.Value)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printExprField(name string, e *Expr) {
	if e == nil {
		p.line(name + ": nil,")
		return
	}
	p.line(name + ": &Expr{")
	p.depth++
	p.printExprBody(e)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printExprBody(e *Expr) {
	p.printTermField("Left", e.Left)
	p.printOpRightSlice("Right", e.Right)
}

func (p *astPrinter) printOpRightSlice(name string, ops []*OpRight) {
	if len(ops) == 0 {
		p.line(name + ": []*OpRight{},")
		return
	}
	p.line(name + ": []*OpRight{")
	p.depth++
	for _, op := range ops {
		p.printOpRight(op)
	}
	p.depth--
	p.line("},")
}

func (p *astPrinter) printOpRight(op *OpRight) {
	if op == nil {
		p.line("nil,")
		return
	}
	p.line("&OpRight{")
	p.depth++
	p.line(fmt.Sprintf("Operator: %q,", op.Operator))
	p.printTermField("Right", op.Right)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printTermField(name string, t *Term) {
	if t == nil {
		p.line(name + ": nil,")
		return
	}
	p.line(name + ": &Term{")
	p.depth++
	p.printTermBody(t)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printTermBody(t *Term) {
	switch {
	case t.Call != nil:
		p.printCallInline(t.Call)
	case t.Ref != nil:
		p.printRefInline(t.Ref)
	case t.String != nil:
		p.line(fmt.Sprintf("String: ptr(%q),", *t.String))
	case t.Int != nil:
		p.line(fmt.Sprintf("Int: ptr(%d),", *t.Int))
	case t.Boolean != nil:
		p.line(fmt.Sprintf("Boolean: ptr(%t),", *t.Boolean))
	}
}

func (p *astPrinter) printCallInline(c *CallExpr) {
	p.line("Call: &CallExpr{")
	p.depth++
	p.line(fmt.Sprintf("Name: %q,", c.Name))
	p.printArgSlice("Args", c.Args)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printRefInline(ref *RefExpr) {
	p.line("Ref: &RefExpr{")
	p.depth++
	p.line(fmt.Sprintf("Root: %q,", ref.Root))
	p.printAccessOpSlice("Chain", ref.Chain)
	p.depth--
	p.line("},")
}

func (p *astPrinter) printAccessOpSlice(name string, ops []*AccessOp) {
	if len(ops) == 0 {
		p.line(name + ": []*AccessOp{},")
		return
	}
	p.line(name + ": []*AccessOp{")
	p.depth++
	for _, op := range ops {
		p.printAccessOp(op)
	}
	p.depth--
	p.line("},")
}

func (p *astPrinter) printAccessOp(op *AccessOp) {
	if op == nil {
		p.line("nil,")
		return
	}
	p.line("&AccessOp{")
	p.depth++
	if op.Property != nil {
		p.line(fmt.Sprintf("Property: ptr(%q),", *op.Property))
	}
	if op.Index != nil {
		p.line(fmt.Sprintf("Index: ptr(%d),", *op.Index))
	}
	p.depth--
	p.line("},")
}
