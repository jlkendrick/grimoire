package types

import (
	"fmt"
	"os"
)

type Config struct {
	Functions []Function `yaml:"functions"`
}

type Function struct {
	Name 	 			   string `yaml:"name"`
	TargetFile 	   string `yaml:"path"`
	TargetFunction string `yaml:"function"`
	Args  		  	 []Arg  `yaml:"args"`
}

func (f Function) String() string {
	return fmt.Sprintf("Function{\n\tName: %s,\n\tTargetFile: %s,\n\tTargetFunction: %s,\n\tArgs: %v\n}", f.Name, f.TargetFile, f.TargetFunction, f.Args)
}

func (f Function) GenerateYAML() string {
	return fmt.Sprintf("  - name: %s\n    file: %s\n    function: %s\n    args: %v\n", f.Name, f.TargetFile, f.TargetFunction, f.Args)
}

func (f Function) LoadSourceCode() ([]byte, error) {
	source_code, err := os.ReadFile(f.TargetFile)
	if err != nil {
		return nil, err
	}
	return source_code, nil
}

type Arg struct {
	Name 		string `yaml:"name"`
	Type 		string `yaml:"type"`
	Default any 	 `yaml:"default"`
}

func (a Arg) String() string {
	return fmt.Sprintf("Arg{\n\t\tName: %s,\n\t\tType: %s,\n\t\tDefault: %v\n\t}", a.Name, a.Type, a.Default)
}