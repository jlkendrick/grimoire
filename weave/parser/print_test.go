package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRitualString(t *testing.T) {
	srcPath := filepath.Join("..", "..", "..", "sample", "test.grm")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatal(err)
	}

	ritual, err := Parse(string(content))
	if err != nil {
		t.Fatal(err)
	}

	got := ritual.String()
	for _, want := range []string{
		"Ritual{",
		`Name: "weather_advice",`,
		"Body: []*Stmt{",
		"Let: &LetStmt{",
		`Ident: "temp",`,
		"Call: &CallExpr{",
		`Name: "classify_temperature",`,
		`Name: "celsius",`,
		"Int: ptr(35),",
		"Ref: &RefExpr{",
		`Root: "temp",`,
		`Operator: "||",`,
		`Operator: "==",`,
		"If: &IfStmt{",
		"Call: &CallExpr{",
		`Name: "warn_frostbite",`,
		`Name: "recommend_shorts",`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output missing %q\n\ngot:\n%s", want, got)
		}
	}
}
