package scroll

import (
	"fmt"
	"encoding/json"

	utils "github.com/jlkendrick/grimoire/internal/utils"
)

type Ritual struct {
	Command string `yaml:"command"`
	Steps   []Step `yaml:"steps"`
}

type Step struct {
	Id 		 string         `yaml:"id"`
	Spell  string         `yaml:"spell"`
	Params map[string]any `yaml:"params"`
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