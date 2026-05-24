package parser

import (
	"fmt"
	"strings"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

type Transpiler struct {
	temp_counter int
}

func (t *Transpiler) nextTempId() string {
	t.temp_counter++
	return fmt.Sprintf("__temp_%d", t.temp_counter)
}

func (t *Transpiler) transpileSteps(stmts []*Stmt) ([]*scroll.Step, error) {
	var yaml_steps []*scroll.Step

	for _, stmt := range stmts {

		if stmt.Let != nil {
			expr_str, hoisted := t.FlattenExpr(stmt.Let.Expr)
			
			for _, hoisted_step := range hoisted {
				fmt.Println(hoisted_step)
			}

			yaml_steps = append(yaml_steps, hoisted...)

			step := &scroll.Step{
				Id: stmt.Let.Ident,
				Value: expr_str,
			}

			fmt.Println(step)

			yaml_steps = append(yaml_steps, step)
		}
	}

	return yaml_steps, nil
}

func (t *Transpiler) FlattenExpr(expr *Expr) (string, []*scroll.Step) {
	if expr == nil {
		return "", nil
	}

	var hoisted []*scroll.Step
	var expr_parts []string
	
	// Flatten the left hand side of the expression
	left_str, left_hoisted := t.FlattenTerm(expr.Left)
	hoisted = append(hoisted, left_hoisted...)
	expr_parts = append(expr_parts, left_str)

	// Flatten the right hand side of the expression
	for _, op := range expr.Right {
		op_str := " " + op.Operator + " "
		op_right_str, op_hoisted := t.FlattenTerm(op.Right)
		op_str += op_right_str
		hoisted = append(hoisted, op_hoisted...)
		expr_parts = append(expr_parts, op_str)
	}

	return strings.Join(expr_parts, ""), hoisted
}

func (t *Transpiler) FlattenTerm(term *Term) (string, []*scroll.Step) {
	// If the term is not a call, just return the stringified term
	if term.Ref != nil {
		return StringifyRef(term.Ref), nil
	}
	if term.String != nil {
		return *term.String, nil
	}
	if term.Int != nil {
		return fmt.Sprintf("%d", *term.Int), nil
	}
	if term.Boolean != nil {
		return fmt.Sprintf("%t", *term.Boolean), nil
	}

	// If the term is a call, flatten the arguments
	if term.Call != nil {

		var hoisted_steps []*scroll.Step
		this_step := &scroll.Step{
			Id: t.nextTempId(),
			Spell: term.Call.Name,
			Params: make(map[string]any),
		}
		hoisted_steps = append(hoisted_steps, this_step)
		
		for _, arg := range term.Call.Args {
			arg_str, arg_hoisted := t.FlattenExpr(arg.Value)
			// If the argument has a function call, we need to hoist it
			// Prepend since we want to evaluate the arguments first
			hoisted_steps = append(arg_hoisted, hoisted_steps...)
			this_step.Params[arg.Name] = arg_str
		}

		return this_step.Id, hoisted_steps
	}

	return "", nil	
}


func StringifyRef(ref *RefExpr) string {
	if ref == nil {
			return ""
	}
	
	var sb strings.Builder
	sb.WriteString(ref.Root)
	
	for _, op := range ref.Chain {
			if op.Property != nil {
					// It's a dot access
					sb.WriteString(".")
					sb.WriteString(*op.Property)
			} else if op.Index != nil {
					// It's a bracket access
					sb.WriteString(fmt.Sprintf("[%d]", *op.Index))
			}
	}
	
	return sb.String()
}
