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
	Name string   `yaml:"name"`
	Target string `yaml:"target"`
}

func (f Function) String() string {
	return fmt.Sprintf("Function{Name: %s, Target: %s}", f.Name, f.Target)
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