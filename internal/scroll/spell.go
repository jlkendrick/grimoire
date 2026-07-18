package scroll

import (
	"fmt"
	"encoding/json"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

type Spell struct {
	Command 	 	string `yaml:"command" json:"command"`
	Path 	   		string `yaml:"path" json:"path"`
	Function 		string `yaml:"function" json:"function"`
	Params  		map[string]any `yaml:"params,omitempty" json:"params,omitempty"`
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