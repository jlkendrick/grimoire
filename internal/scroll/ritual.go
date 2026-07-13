package scroll

import (
	"encoding/json"
	"fmt"

	utils "github.com/jlkendrick/grimoire/internal/utils"
)

type Ritual struct {
	Command string `yaml:"command"`
	Steps   []Step `yaml:"steps"`
}

// Step is one of four kinds: spell-step (Spell set), if-step (If set),
// let-step (Let set, binding a name to the result of an expression), or
// print-step (Print set, declaring a ritual output — the expression's
// value goes to the frontend's presenter). The reconciler enforces
// mutual exclusivity and the other shape rules (no id on if-step,
// non-empty then, no spell+if mixing, first step is a spell so CLI
// flags can be derived).
//
// The newer fields (If/Then/Else/Let/Value/Print) carry json:",omitempty"
// so adding them doesn't perturb the JSON hash of pre-existing spell-only
// rituals.
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
