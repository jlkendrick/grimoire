package scroll

import (
	"os"
	"fmt"
	"encoding/json"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

type Spell struct {
	Command 	 	string `yaml:"command" json:"command"`
	Path 	   		string `yaml:"path" json:"path"`
	Function 		string `yaml:"function" json:"function"`
	Params  		[]Param  `yaml:"params,omitempty" json:"params,omitempty"`
	Interpreter string `yaml:"interpreter,omitempty" json:"interpreter,omitempty"`

	ScrollPath 	string `yaml:"-" json:"-"`
	AbsPath 		string `yaml:"-" json:"-"`
}

func GenerateMinimalSpellFromFunctionDescriptor(function_descriptor descriptor.FunctionDescriptor) (Spell, error) {
	return Spell{
		Command: function_descriptor.CommandName,
		Path: function_descriptor.RelPathToSourceFile,
		Function: function_descriptor.FunctionName,
	}, nil
}

func (s Spell) String() string {
	return fmt.Sprintf("Spell{\n\tCommand: %s,\n\tPath: %s,\n\tFunction: %s,\n\tParams: %v,\n\tInterpreter: %s\n}", s.Command, s.Path, s.Function, s.Params, s.Interpreter)
}

func (s Spell) GenerateYAML() string {
	return fmt.Sprintf("  - name: %s\n    file: %s\n    function: %s\n    args: %v\n    interpreter: %s\n", s.Command, s.Path, s.Function, s.Params, s.Interpreter)
}

func (s Spell) LoadSourceCode() ([]byte, error) {
	p, err := utils.ExpandUserPath(s.Path)
	if err != nil {
		return nil, fmt.Errorf("error resolving path: %w", err)
	}
	source_code, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("error loading source code: %v", err)
	}
	return source_code, nil
}

func (s *Spell) Hash() (string, error) {
	canonical, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("error marshalling spell: %v", err)
	}
	hash, err := utils.HashStr(canonical)
	if err != nil {
		return "", fmt.Errorf("error hashing spell: %v", err)
	}
	return hash, nil
}

type Param struct {
	Name 		string `yaml:"name"`
	Type 		string `yaml:"type"`
	Default any 	 `yaml:"default,omitempty"`
}

// UnmarshalYAML stringifies Default so the descriptor IR carries a single
// canonical form. Typed coercion happens at the use site (cobra flag
// construction).
func (p *Param) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawParam Param
	var tmp rawParam
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	*p = Param(tmp)

	if p.Default == nil {
		return nil
	}
	if _, ok := p.Default.(string); !ok {
		p.Default = fmt.Sprint(p.Default)
	}
	return nil
}

func (p Param) String() string {
	return fmt.Sprintf("Param{\n\t\tName: %s,\n\t\tType: %s,\n\t\tDefault: %v\n\t}", p.Name, p.Type, p.Default)
}