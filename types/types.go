package types

import (
	"os"
	"fmt"
	"strconv"
)

type Config struct {
	Functions []Function `yaml:"functions"`
}

type Function struct {
	Name 	 			   string `yaml:"name"`
	TargetFile 	   string `yaml:"path"`
	TargetFunction string `yaml:"function,omitempty"`
	Args  		  	 []Arg  `yaml:"args,omitempty"`
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

func (a *Arg) CastAndSetDefault() error {
	// Cast the default values to the appropriate type
	switch a.Type {

	case "string":
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