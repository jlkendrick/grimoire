package config

import (
	"os"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Functions []Function `yaml:"functions"`
}

type Function struct {
	Name 	 string `yaml:"name"`
	Target string `yaml:"target"`
	Args   []Arg  `yaml:"args"`
}

func (f Function) String() string {
	return fmt.Sprintf("Function{\n\tName: %s,\n\tTarget: %s,\n\tArgs: %v\n}", f.Name, f.Target, f.Args)
}

func (f Function) GenerateYAML() string {
	return fmt.Sprintf("  - name: %s\n    target: %s\n    args: %v\n", f.Name, f.Target, f.Args)
}

// func (f Function) ValidateArgs(args map[string]any) error {
// 	for _, arg := range f.Args {
// 		if _, ok := args[arg.Name]; !ok {
// 			if arg.Default == nil {
// 				return fmt.Errorf("argument %s is required", arg.Name)
// 			}
// 			args[arg.Name] = arg.Default
// 		} else {
// 			switch arg.Type {
// 			case "int":
// 				if _, ok := args[arg.Name].(int); !ok {
// 					return fmt.Errorf("argument %s must be an integer", arg.Name)
// 				}
// 			case "float":
// 				if _, ok := args[arg.Name].(float64); !ok {
// 					return fmt.Errorf("argument %s must be a float", arg.Name)
// 				}
// 			case "string":
// 				if _, ok := args[arg.Name].(string); !ok {
// 					return fmt.Errorf("argument %s must be a string", arg.Name)
// 				}
// 			case "bool":
// 				if _, ok := args[arg.Name].(bool); !ok {
// 					return fmt.Errorf("argument %s must be a boolean", arg.Name)
// 				}
// 			default:
// 				return fmt.Errorf("argument %s has an invalid type", arg.Name)
// 			}
// 		}
// 	}
// 	return nil
// }

type Arg struct {
	Name 		string `yaml:"name"`
	Type 		string `yaml:"type"`
	Default any 	 `yaml:"default"`
}

func (a Arg) String() string {
	return fmt.Sprintf("Arg{\n\t\tName: %s,\n\t\tType: %s,\n\t\tDefault: %v\n\t}", a.Name, a.Type, a.Default)
}

type LanguageAnalyzer interface {
	ExtractSignature(function Function) ([]Arg, error)
}

type PythonAnalyzer struct {
}

func (a *PythonAnalyzer) ExtractSignature(function Function) ([]Arg, error) {
	return function.Args, nil
}

type ConfigGenerator struct {
	ConfigPath string
	Functions  []Function
}


func ParseUserYAML(path string) ([]Function, error) {
	yamlStr, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var user_config Config
	if err := yaml.Unmarshal([]byte(yamlStr), &user_config); err != nil {
		return nil, err
	}

	return user_config.Functions, nil
}

func (g *ConfigGenerator) GenerateTypedYAML() error {
	yaml_str := ""

	for _, function := range g.Functions {
			var analyzer LanguageAnalyzer
	
			target_parts := strings.Split(function.Target, ":")
			if len(target_parts) == 0 {
				return fmt.Errorf("invalid target: %s", function.Target)
			}
	
			file_path := target_parts[0]
			if !strings.HasSuffix(file_path, ".py") {
				return fmt.Errorf("unsupported file type: %s", file_path)
			}
	
			file_extension := strings.Split(file_path, ".")[1]
			switch file_extension {
			case "py":
				analyzer = &PythonAnalyzer{}
			default:
				return fmt.Errorf("unsupported file extension: %s", file_extension)
			}

		args, err := analyzer.ExtractSignature(function)
		if err != nil {
			return err
		}

		function.Args = args

		func_yaml, err := yaml.Marshal(function)
		if err != nil {
			return err
		}
		yaml_str += string(func_yaml)
	}

	err := os.WriteFile(g.ConfigPath, []byte(yaml_str), 0644)
	if err != nil {
		return err
	}

	return nil
}