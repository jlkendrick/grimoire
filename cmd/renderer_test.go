package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

func TestRenderer_PlainSequential(t *testing.T) {
	var out, errB bytes.Buffer
	r := newRenderer(&out, &errB, false, 2)

	r.spellStart(1, "fetch")
	r.spellStderr(1, "chatter line")
	r.spellFinish(1, "fetch", map[string]any{"ok": true}, "python 3.12", nil)
	r.spellStart(2, "double")
	r.spellFinish(2, "double", 6.0, "python 3.12", nil)
	if err := r.present(map[string]any{"v": 6.0}); err != nil {
		t.Fatal(err)
	}
	r.footer(1.5)

	e := stripANSI(errB.String())
	for _, want := range []string{
		"step 1/2 · fetch",
		"step 2/2 · double",
		`{"ok":true}`, // finish preview of the return value
		"1.50s · python 3.12",
	} {
		if !strings.Contains(e, want) {
			t.Errorf("stderr missing %q:\n%s", want, e)
		}
	}
	if strings.Contains(e, "chatter line") {
		t.Error("plain mode must swallow stderr chatter")
	}
	if strings.Contains(e, "✓") {
		t.Error("sequential completions must use the unnamed style")
	}
	if got := out.String(); got != "{\"v\":6}\n" {
		t.Errorf("stdout = %q, want only the declared print", got)
	}
}

func TestRenderer_GroupedCompletionsAreNamed(t *testing.T) {
	var out, errB bytes.Buffer
	r := newRenderer(&out, &errB, false, 2)

	r.spellStart(1, "alpha")
	r.spellStart(2, "beta") // overlap: both spells are grouped from here on
	r.spellFinish(1, "alpha", 1.0, "", nil)
	// beta finishes alone, but it ran concurrently — named style sticks.
	r.spellFinish(2, "beta", 2.0, "", nil)

	e := stripANSI(errB.String())
	if !strings.Contains(e, "✓ alpha → 1") || !strings.Contains(e, "✓ beta → 2") {
		t.Errorf("grouped completions must be named:\n%s", e)
	}
}

func TestRenderer_ErrorCompletion(t *testing.T) {
	var out, errB bytes.Buffer
	r := newRenderer(&out, &errB, false, 1)

	r.spellStart(1, "boom")
	r.spellFinish(1, "boom", nil, "", &testError{})

	e := stripANSI(errB.String())
	if !strings.Contains(e, "✗ boom") {
		t.Errorf("error completion missing:\n%s", e)
	}
}

type testError struct{}

func (*testError) Error() string { return "kaput" }
