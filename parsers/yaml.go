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