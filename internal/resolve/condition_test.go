package resolve_test

import (
	"strings"
	"testing"

	resolve "github.com/jlkendrick/grimoire/internal/resolve"
)

func mustParse(t *testing.T, src string) resolve.Expr {
	t.Helper()
	expr, err := resolve.ParseCondition(src)
	if err != nil {
		t.Fatalf("ParseCondition(%q): %v", src, err)
	}
	return expr
}

func TestParseCondition_Literals(t *testing.T) {
	cases := []struct {
		src  string
		want any
	}{
		{"true", true},
		{"false", false},
		{"null", nil},
		{"42", 42.0},
		{"-3.14", -3.14},
		{`"hello"`, "hello"},
		{`"with space"`, "with space"},
	}
	for _, c := range cases {
		expr := mustParse(t, c.src)
		lit, ok := expr.(resolve.LiteralExpr)
		if !ok {
			t.Errorf("%q: expected LiteralExpr, got %T", c.src, expr)
			continue
		}
		if lit.Value != c.want {
			t.Errorf("%q: got %v, want %v", c.src, lit.Value, c.want)
		}
	}
}

func TestParseCondition_References(t *testing.T) {
	cases := []string{
		"check",
		"check.ok",
		"check.user.name",
		"check[0]",
		"check[0].field",
		"check.list[2].name",
	}
	for _, src := range cases {
		expr := mustParse(t, src)
		ref, ok := expr.(resolve.ReferenceExpr)
		if !ok {
			t.Errorf("%q: expected ReferenceExpr, got %T", src, expr)
			continue
		}
		if ref.Path != src {
			t.Errorf("%q: Path = %q", src, ref.Path)
		}
	}
}

func TestParseCondition_Errors(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"&&",
		"a &&",
		"a > ",
		"(a",
		"a)",
		`"unterminated`,
		"a > > b",
	}
	for _, src := range cases {
		if _, err := resolve.ParseCondition(src); err == nil {
			t.Errorf("ParseCondition(%q): expected error, got nil", src)
		}
	}
}

func TestEvaluateCondition_BareBool(t *testing.T) {
	bindings := map[string]any{"check": map[string]any{"ok": true}}
	expr := mustParse(t, "check.ok")
	got, err := resolve.EvaluateCondition(expr, bindings)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !got {
		t.Errorf("got false, want true")
	}
}

func TestEvaluateCondition_StrictBoolNoCoercion(t *testing.T) {
	// A bare reference resolving to a number should ERROR — no truthy
	// coercion. Users must compare explicitly.
	bindings := map[string]any{"count": 5.0}
	expr := mustParse(t, "count")
	if _, err := resolve.EvaluateCondition(expr, bindings); err == nil {
		t.Errorf("expected error for non-bool bare reference, got nil")
	}
}

func TestEvaluateCondition_Comparisons(t *testing.T) {
	bindings := map[string]any{
		"count":  3.0,
		"status": "ready",
		"empty":  nil,
	}
	cases := []struct {
		src  string
		want bool
	}{
		{"count > 0", true},
		{"count > 5", false},
		{"count == 3", true},
		{"count != 3", false},
		{"count >= 3", true},
		{"count <= 2", false},
		{`status == "ready"`, true},
		{`status != "error"`, true},
		{"empty == null", true},
		{"empty != null", false},
		{"count > 0 && count < 10", true},
		{"count > 5 || count < 10", true},
		{"!(count > 5)", true},
		{"(count > 0) && (count < 10)", true},
	}
	for _, c := range cases {
		expr := mustParse(t, c.src)
		got, err := resolve.EvaluateCondition(expr, bindings)
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q: got %v, want %v", c.src, got, c.want)
		}
	}
}

func TestEvaluateCondition_OrderingRequiresNumeric(t *testing.T) {
	bindings := map[string]any{"name": "foo"}
	expr := mustParse(t, `name > "bar"`)
	if _, err := resolve.EvaluateCondition(expr, bindings); err == nil {
		t.Errorf("expected error comparing strings with >, got nil")
	}
}

func TestEvaluateCondition_LogicalRequiresBool(t *testing.T) {
	// `count && true` is a type error — logical ops require bool operands.
	bindings := map[string]any{"count": 3.0}
	expr := mustParse(t, "count && true")
	if _, err := resolve.EvaluateCondition(expr, bindings); err == nil {
		t.Errorf("expected error for non-bool && operand, got nil")
	}
}

func TestEvaluateCondition_MissingBinding(t *testing.T) {
	bindings := map[string]any{}
	expr := mustParse(t, "missing.field")
	_, err := resolve.EvaluateCondition(expr, bindings)
	if err == nil {
		t.Fatalf("expected error for missing binding, got nil")
	}
	if !strings.Contains(err.Error(), "missing") && !strings.Contains(err.Error(), "scope") {
		t.Errorf("error message %q should mention the missing reference", err.Error())
	}
}

func TestEvaluateCondition_ShortCircuit(t *testing.T) {
	// `false && <undefined>` should short-circuit and NOT error.
	bindings := map[string]any{}
	expr := mustParse(t, "false && missing")
	got, err := resolve.EvaluateCondition(expr, bindings)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got {
		t.Errorf("got true, want false")
	}
	// And `true || <undefined>` should short-circuit to true.
	expr2 := mustParse(t, "true || missing")
	got2, err := resolve.EvaluateCondition(expr2, bindings)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !got2 {
		t.Errorf("got false, want true")
	}
}

func TestEvaluateCondition_Precedence(t *testing.T) {
	// `&&` binds tighter than `||`: false || true && false  →  false || (true && false)  →  false
	bindings := map[string]any{}
	expr := mustParse(t, "false || true && false")
	got, err := resolve.EvaluateCondition(expr, bindings)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if got {
		t.Errorf("got true, want false (&& should bind tighter than ||)")
	}
}
