package weave

import (
	"fmt"
	"strconv"
	"strings"

	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	ast "github.com/jlkendrick/grimoire/weave/parser"
)

type Transpiler struct {
	temp_counter int
}

// FlatValue carries the result of flattening a term/expression in two forms.
// Val is the natural Go value (int, bool, string for literals; the resolved
// source fragment for refs and call binding ids) and is what gets stored in a
// Step.Params map so the runtime sees typed payloads. Src is the
// expression-grammar source fragment for splicing into Step.If / Step.Value
// strings — string literals are re-quoted so the grammar parses them as
// literals rather than references.
type FlatValue struct {
	Val any
	Src string
}

func (t *Transpiler) TranspileRitual(ritual *ast.Ritual) (scroll.Ritual, error) {
	steps, err := t.transpileSteps(ritual.Body)
	if err != nil {
		return scroll.Ritual{}, err
	}
	return scroll.Ritual{
		Command: ritual.Name,
		Steps:   steps,
	}, nil
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

			flat, hoisted := t.FlattenExpr(stmt.Let.Expr)

			yaml_steps = append(yaml_steps, hoisted...)

			step := scroll.Step{
				Let:   stmt.Let.Ident,
				Value: flat.Src,
			}

			yaml_steps = append(yaml_steps, step)
		}

		if stmt.If != nil {
			condition, condition_hoisted := t.FlattenExpr(stmt.If.Condition)

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
				If:   condition.Src,
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

func (t *Transpiler) nextTempId() string {
	t.temp_counter++
	return fmt.Sprintf("__temp_%d", t.temp_counter)
}

// FlattenExpr lowers an AST expression to a FlatValue and any steps that
// must be hoisted before its containing step. If the expression has no
// binary operators, the left term's FlatValue is returned as-is so literals
// keep their natural Go types. Otherwise the term Src fragments are joined
// into an expression-grammar string, and both Val and Src hold that string.
func (t *Transpiler) FlattenExpr(expr *ast.Expr) (FlatValue, []scroll.Step) {
	if expr == nil {
		return FlatValue{Val: "", Src: ""}, nil
	}

	left, hoisted := t.FlattenTerm(expr.Left)
	if len(expr.Right) == 0 {
		return left, hoisted
	}

	var sb strings.Builder
	sb.WriteString(left.Src)
	for _, op := range expr.Right {
		right, op_hoisted := t.FlattenTerm(op.Right)
		hoisted = append(hoisted, op_hoisted...)
		sb.WriteString(" ")
		sb.WriteString(op.Operator)
		sb.WriteString(" ")
		sb.WriteString(right.Src)
	}

	joined := sb.String()
	return FlatValue{Val: joined, Src: joined}, hoisted
}

// FlattenTerm lowers an AST term to a FlatValue. Literals keep their natural
// Go types in Val and use grammar-quoted forms in Src (so a string literal
// spliced into an expression is parsed as a literal, not a reference). Refs
// and call ids use the same string in both fields — they are valid references
// in the expression grammar and also valid binding-name payloads in Params.
func (t *Transpiler) FlattenTerm(term *ast.Term) (FlatValue, []scroll.Step) {
	if term.Ref != nil {
		s := StringifyRef(term.Ref)
		return FlatValue{Val: s, Src: s}, nil
	}
	if term.String != nil {
		// The default participle lexer's @String token includes the surrounding
		// quotes, so the raw token is already a valid grammar fragment for Src.
		// Val needs the unquoted content so Params payloads carry the natural
		// string value rather than a re-quoted form.
		raw := *term.String
		val, err := strconv.Unquote(raw)
		if err != nil {
			val = raw
		}
		return FlatValue{Val: val, Src: raw}, nil
	}
	if term.Int != nil {
		return FlatValue{Val: *term.Int, Src: strconv.Itoa(*term.Int)}, nil
	}
	if term.Boolean != nil {
		return FlatValue{Val: *term.Boolean, Src: strconv.FormatBool(*term.Boolean)}, nil
	}

	// If the term is a call, flatten the arguments. The call appears nested
	// inside an expression, so we need a temp id to substitute back into the
	// flattened expression string. The id becomes both the Params payload
	// (engine resolves it as a binding reference) and the expression source.
	if term.Call != nil {
		id, hoisted := t.FlattenCall(term.Call, t.nextTempId())
		return FlatValue{Val: id, Src: id}, hoisted
	}

	return FlatValue{Val: "", Src: ""}, nil
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
// referenced — no Id field is set on the emitted step. Argument values are
// stored as their natural Go types in Params; only when an arg is a composed
// expression does it land as a string (the joined expression source).
func (t *Transpiler) FlattenCall(call *ast.CallExpr, id string) (string, []scroll.Step) {
	spellName := call.Name
	if call.Module != "" {
		spellName = call.Module + "." + call.Name
	}
	this_step := scroll.Step{
		Id:     id,
		Spell:  spellName,
		Params: make(map[string]any),
	}

	var arg_hoisted_all []scroll.Step
	for _, arg := range call.Args {
		flat, arg_hoisted := t.FlattenExpr(arg.Value)
		arg_hoisted_all = append(arg_hoisted_all, arg_hoisted...)
		this_step.Params[arg.Name] = flat.Val
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
			sb.WriteString(".")
			sb.WriteString(*op.Property)
		} else if op.Index != nil {
			sb.WriteString(fmt.Sprintf("[%d]", *op.Index))
		}
	}

	return sb.String()
}
