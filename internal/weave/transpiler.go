package weave

import (
	"fmt"
	"strings"

	ast "github.com/jlkendrick/grimoire/internal/weave/parser"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

type Transpiler struct {
	temp_counter int
}

func (t *Transpiler) nextTempId() string {
	t.temp_counter++
	return fmt.Sprintf("__temp_%d", t.temp_counter)
}

func (t *Transpiler) transpileSteps(stmts []*ast.Stmt) ([]scroll.Step, error) {
	var yaml_steps []scroll.Step

	for _, stmt := range stmts {

		if stmt.Let != nil {
			// Bare call on the RHS — emit a single spell-step with the let's
			// identifier as the step id, rather than hoisting an anonymous
			// step and binding it via a let-step.
			if call := bareCall(stmt.Let.Expr); call != nil {
				_, hoisted := t.FlattenCall(call, stmt.Let.Ident)
				yaml_steps = append(yaml_steps, hoisted...)
				continue
			}

			expr_str, hoisted := t.FlattenExpr(stmt.Let.Expr)

			yaml_steps = append(yaml_steps, hoisted...)

			step := scroll.Step{
				Id: stmt.Let.Ident,
				Value: expr_str,
			}

			yaml_steps = append(yaml_steps, step)
		}

		if stmt.If != nil {
			condition_str, condition_hoisted := t.FlattenExpr(stmt.If.Condition)

			yaml_steps = append(yaml_steps, condition_hoisted...)

			true_steps, err := t.transpileSteps(stmt.If.TrueBlock)
			if err != nil {
				return nil, err
			}
			false_steps, err := t.transpileSteps(stmt.If.ElseBlock)
			if err != nil {
				return nil, err
			}

			step := scroll.Step{
				If: condition_str,
				Then: true_steps,
				Else: false_steps,
			}

			yaml_steps = append(yaml_steps, step)
		}

		if stmt.Call != nil {
			_, call_hoisted := t.FlattenCall(stmt.Call, "")

			yaml_steps = append(yaml_steps, call_hoisted...)
		}
	}

	return yaml_steps, nil
}

func (t *Transpiler) FlattenExpr(expr *ast.Expr) (string, []scroll.Step) {
	if expr == nil {
		return "", nil
	}

	var hoisted []scroll.Step
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

func (t *Transpiler) FlattenTerm(term *ast.Term) (string, []scroll.Step) {
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

	// If the term is a call, flatten the arguments. The call appears nested
	// inside an expression, so we need a temp id to substitute back into the
	// flattened expression string.
	if term.Call != nil {
		return t.FlattenCall(term.Call, t.nextTempId())
	}

	return "", nil
}

// bareCall returns the call if expr is exactly a single call term with no
// binary operators, nil otherwise. Used to skip unnecessary hoisting when a
// let's RHS or a top-level statement is just a direct call.
func bareCall(expr *ast.Expr) *ast.CallExpr {
	if expr == nil || len(expr.Right) != 0 || expr.Left == nil {
		return nil
	}
	return expr.Left.Call
}

// FlattenCall emits a spell-step for the given call and returns its id along
// with the hoisted steps (which include the call's own step plus any
// argument-call hoisting). An empty id means the step's output is not
// referenced — no Id field is set on the emitted step.
func (t *Transpiler) FlattenCall(call *ast.CallExpr, id string) (string, []scroll.Step) {
	this_step := scroll.Step{
		Id: id,
		Spell: call.Name,
		Params: make(map[string]any),
	}

	var arg_hoisted_all []scroll.Step
	for _, arg := range call.Args {
		arg_str, arg_hoisted := t.FlattenExpr(arg.Value)
		// Evaluate argument-call hoisting before this step.
		arg_hoisted_all = append(arg_hoisted_all, arg_hoisted...)
		this_step.Params[arg.Name] = arg_str
	}

	return id, append(arg_hoisted_all, this_step)
}


func StringifyRef(ref *ast.RefExpr) string {
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
