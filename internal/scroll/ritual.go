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

// Step is either a spell-step (Spell set) or an if-step (If set). The
// reconciler enforces mutual exclusivity and the other shape rules
// (no id on if-step, non-empty then, if-step not in first position).
//
// New fields (If/Then/Else) carry json:",omitempty" so adding them
// doesn't perturb the JSON hash of pre-existing spell-only rituals.
type Step struct {
	Id     string         `yaml:"id,omitempty"`
	Spell  string         `yaml:"spell,omitempty"`
	Params map[string]any `yaml:"params,omitempty"`

	If   string `yaml:"if,omitempty" json:",omitempty"`
	Then []Step `yaml:"then,omitempty" json:",omitempty"`
	Else []Step `yaml:"else,omitempty" json:",omitempty"`
}

func (s Step) Kind() string {
	if s.If != "" {
		return "if"
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
