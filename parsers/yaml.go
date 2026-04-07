package parsers

import (
	"os"
	"fmt"

	"github.com/goccy/go-yaml"
)


type Config struct {
	Functions []Function `yaml:"functions"`
}


type Function struct {
	Name 	 string `yaml:"name"`
	Target string `yaml:"target"`
	Args   []Arg   `yaml:"args"`
}
func (f Function) String() string {
	return fmt.Sprintf("Function{\n\tName: %s,\n\tTarget: %s,\n\tArgs: %v\n}", f.Name, f.Target, f.Args)
}
func (f Function) ValidateArgs(args map[string]any) error {
	for _, arg := range f.Args {
		if _, ok := args[arg.Name]; !ok {
			if arg.Default == nil {
				return fmt.Errorf("argument %s is required", arg.Name)
			}
			args[arg.Name] = arg.Default
		} else {
			switch arg.Type {
			case "int":
				if _, ok := args[arg.Name].(int); !ok {
					return fmt.Errorf("argument %s must be an integer", arg.Name)
				}
			case "float":
				if _, ok := args[arg.Name].(float64); !ok {
					return fmt.Errorf("argument %s must be a float", arg.Name)
				}
			case "string":
				if _, ok := args[arg.Name].(string); !ok {
					return fmt.Errorf("argument %s must be a string", arg.Name)
				}
			case "bool":
				if _, ok := args[arg.Name].(bool); !ok {
					return fmt.Errorf("argument %s must be a boolean", arg.Name)
				}
			default:
				return fmt.Errorf("argument %s has an invalid type", arg.Name)
			}
		}
	}
	return nil
}


type Arg struct {
	Name 		string `yaml:"name"`
	Type 		string `yaml:"type"`
	Default any 	 `yaml:"default"`
}
func (a Arg) String() string {
	return fmt.Sprintf("Arg{\n\t\tName: %s,\n\t\tType: %s,\n\t\tDefault: %v\n\t}", a.Name, a.Type, a.Default)
}


func ParseYAML(path string) ([]Function, error) {
	yamlStr, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal([]byte(yamlStr), &config); err != nil {
		return nil, err
	}

	return config.Functions, nil
}