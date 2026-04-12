package types

import (
	"os"
	"fmt"
	"strconv"

	utils "github.com/jlkendrick/grimoire/utils"

	"github.com/goccy/go-yaml"
)


type Config struct {
	Functions []Function `yaml:"functions"`
}


func (c *Config) Write(path string) error {
	yaml_content, err := yaml.MarshalWithOptions(c, 
		yaml.Indent(2),
		yaml.IndentSequence(true),
	)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, yaml_content, 0644)
	if err != nil {
		return err
	}
	return nil
}

type Function struct {
	Name 	 			   string `yaml:"name"`
	TargetFile 	   string `yaml:"path"`
	TargetFunction string `yaml:"function,omitempty"`
	Args  		  	 []Arg  `yaml:"args,omitempty"`
	Interpreter 	 string `yaml:"interpreter,omitempty"`
}

func (f Function) String() string {
	return fmt.Sprintf("Function{\n\tName: %s,\n\tTargetFile: %s,\n\tTargetFunction: %s,\n\tArgs: %v,\n\tInterpreter: %s\n}", f.Name, f.TargetFile, f.TargetFunction, f.Args, f.Interpreter)
}

func (f Function) GenerateYAML() string {
	return fmt.Sprintf("  - name: %s\n    file: %s\n    function: %s\n    args: %v\n    interpreter: %s\n", f.Name, f.TargetFile, f.TargetFunction, f.Args, f.Interpreter)
}

func (f Function) LoadSourceCode() ([]byte, error) {
	p, err := utils.ExpandUserPath(f.TargetFile)
	if err != nil {
		return nil, fmt.Errorf("error resolving path: %w", err)
	}
	source_code, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("error loading source code: %v", err)
	}
	return source_code, nil
}

type Arg struct {
	Name 		string `yaml:"name"`
	Type 		string `yaml:"type"`
	Default any 	 `yaml:"default,omitempty"`
}

// UnmarshalYAML normalizes the concrete type of Default when YAML has already
// parsed it into a scalar (e.g. uint8(3) vs int(3)). We intentionally do not
// coerce string defaults like "1" into numeric types here, because user config
// may represent defaults as strings prior to typed casting/validation.
func (a *Arg) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawArg Arg
	var tmp rawArg
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	*a = Arg(tmp)

	// Normalize only when YAML produced a non-string scalar.
	switch a.Type {
	case "int":
		switch v := a.Default.(type) {
		case int:
			// ok
		case int8:
			a.Default = int(v)
		case int16:
			a.Default = int(v)
		case int32:
			a.Default = int(v)
		case int64:
			a.Default = int(v)
		case uint:
			a.Default = int(v)
		case uint8:
			a.Default = int(v)
		case uint16:
			a.Default = int(v)
		case uint32:
			a.Default = int(v)
		case uint64:
			a.Default = int(v)
		}
	case "float":
		switch v := a.Default.(type) {
		case float64:
			// ok
		case float32:
			a.Default = float64(v)
		case int:
			a.Default = float64(v)
		case int8:
			a.Default = float64(v)
		case int16:
			a.Default = float64(v)
		case int32:
			a.Default = float64(v)
		case int64:
			a.Default = float64(v)
		case uint:
			a.Default = float64(v)
		case uint8:
			a.Default = float64(v)
		case uint16:
			a.Default = float64(v)
		case uint32:
			a.Default = float64(v)
		case uint64:
			a.Default = float64(v)
		}
	}

	return nil
}

func (a Arg) String() string {
	return fmt.Sprintf("Arg{\n\t\tName: %s,\n\t\tType: %s,\n\t\tDefault: %v\n\t}", a.Name, a.Type, a.Default)
}

func (a *Arg) CastAndSetDefault() error {
	// Cast the default values to the appropriate type
	switch a.Type {

	case "string", "str":
		a.Default = a.Default.(string)

	case "int":
		int_default, err := strconv.Atoi(a.Default.(string))
		if err != nil {
			return fmt.Errorf("error converting default value to int: %v", err)
		}
		a.Default = int_default

	case "bool":
		bool_default, err := strconv.ParseBool(a.Default.(string))
		if err != nil {
			return fmt.Errorf("error converting default value to bool: %v", err)
		}
		a.Default = bool_default

	case "float":
		float_default, err := strconv.ParseFloat(a.Default.(string), 64)
		if err != nil {
			return fmt.Errorf("error converting default value to float: %v", err)
		}
		a.Default = float_default

	default:
		return fmt.Errorf("unsupported type: %s", a.Type)
	}
	
	return nil
}