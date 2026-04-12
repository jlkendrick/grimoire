package config

import (
	"os"
	"fmt"
	"strings"

	types "github.com/jlkendrick/grimoire/types"
	parsers "github.com/jlkendrick/grimoire/parsers"

	"github.com/goccy/go-yaml"
)

type ConfigGenerator struct {
	ConfigPath   string
	Config 	     *types.Config
	ManifestYAML string
}

// Parse the user's configuration file
func ParseConfig(path string) (*types.Config, error) {
	yamlStr, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config types.Config
	if err := yaml.Unmarshal([]byte(yamlStr), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Generate the typed YAML file, extracting the function signatures from the source code for validation
func (g *ConfigGenerator) GenerateManifestYAML() (string, error) {

	// For each function in the configuration, generate the typed YAML
	for i, function := range g.Config.Functions {
		if function.TargetFunction == "" {
			continue
		}

		var analyzer parsers.LanguageAnalyzer
	
		if !strings.Contains(function.TargetFile, ".") {
			return "", fmt.Errorf("no file extension found: %s", function.TargetFile)
		}

		// Determine the file extension and use the appropriate analyzer
		file_extensions := strings.Split(function.TargetFile, ".")
		file_extension := file_extensions[len(file_extensions)-1]
		switch file_extension {
		case "py":
			analyzer = &parsers.PythonAnalyzer{}
		default:
			return "", fmt.Errorf("unsupported file extension: %s", file_extension)
		}

		// Extract the function signature from the source code
		args, err := analyzer.ExtractSignature(function)
		if err != nil {
			return "", err
		}

		// Cast the default values to the appropriate type
		for j := range args {
			if args[j].Default != nil {
				err := args[j].CastAndSetDefault()
				if err != nil {
					return "", err
				}
			}
		}

		// Update the function with the extracted and casted arguments
		g.Config.Functions[i].Args = args
	}

	// Marshal the config to YAML
	manifest_yaml, err := yaml.MarshalWithOptions(g.Config, 
		yaml.Indent(2),
		yaml.IndentSequence(true),
	)
	if err != nil {
		return "", err
	}

	return string(manifest_yaml), nil
}

func (g *ConfigGenerator) WriteManifestYAML(manifest_yaml string) error {
	err := os.WriteFile(g.ConfigPath, []byte(manifest_yaml), 0644)
	if err != nil {
		return fmt.Errorf("error writing manifest YAML: %v", err)
	}
	return nil
}