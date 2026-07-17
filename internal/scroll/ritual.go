package scroll

import (
	"encoding/json"
	"fmt"

	utils "github.com/jlkendrick/grimoire/internal/utils"
)

type Ritual struct {
	Command string `yaml:"command"`
	Steps   []Step `yaml:"steps"`

	// Mode is the data-flow contract of the ritual's top-level scope:
	// "pipe" (the default) chains steps implicitly in declaration order;
	// "graph" derives a dependency DAG from declared references and runs
	// independent steps concurrently. The value is validated at scroll
	// load by the graph builder.
	Mode string `yaml:"mode,omitempty" json:",omitempty"`
}

// Step is one of four kinds: spell-step (Spell set), if-step (If set),
// let-step (Let set, binding a name to the result of an expression), or
// print-step (Print set, declaring a ritual output — the expression's
// value goes to the frontend's presenter). The reconciler enforces
// mutual exclusivity and the other shape rules (no id on if-step,
// non-empty then, no spell+if mixing, first step is a spell so CLI
// flags can be derived).
//
// The newer fields (If/Then/Else/Let/Value/Print/Mode) carry
// json:",omitempty" so adding them doesn't perturb the JSON hash of
// pre-existing spell-only rituals.
type Step struct {
	Id     string         `yaml:"id,omitempty"`
	Spell  string         `yaml:"spell,omitempty"`
	Params map[string]any `yaml:"params,omitempty"`

	If   string `yaml:"if,omitempty" json:",omitempty"`
	Then []Step `yaml:"then,omitempty" json:",omitempty"`
	Else []Step `yaml:"else,omitempty" json:",omitempty"`

	Let   string `yaml:"let,omitempty" json:",omitempty"`
	Value string `yaml:"value,omitempty" json:",omitempty"`

	Print string `yaml:"print,omitempty" json:",omitempty"`

	// Mode is valid on if-steps only: it overrides the mode of both
	// branch scopes, which otherwise inherit the enclosing scope's.
	// Mode is an attribute of the scope's owner, not a step kind —
	// which is why it cannot appear mid-scope or twice per scope.
	Mode string `yaml:"mode,omitempty" json:",omitempty"`
}

func (s Step) Kind() string {
	if s.Let != "" {
		return "let"
	}
	if s.If != "" {
		return "if"
	}
	if s.Print != "" {
		return "print"
	}
	return "spell"
}

func (r *Ritual) Hash() (string, error) {
	canonical, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("error marshalling ritual: %v", err)
	}
	hash, err := utils.HashStr(canonical)
	if err != nil {
		return "", fmt.Errorf("error hashing ritual: %v", err)
	}
	return hash, nil
}
