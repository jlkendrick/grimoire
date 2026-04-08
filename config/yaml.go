package config

import (
	"os"
	"fmt"
	"strconv"
	"strings"

	types "github.com/jlkendrick/sigil/types"
	parsers "github.com/jlkendrick/sigil/parsers"

	"github.com/goccy/go-yaml"
)

type ConfigGenerator struct {
	ConfigPath string
	Config 	   *types.Config
}

// Parse the user's configuration file
func ParseUserConfig(path string) (*types.Config, error) {
	yamlStr, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var user_config types.Config
	if err := yaml.Unmarshal([]byte(yamlStr), &user_config); err != nil {
		return nil, err
	}

	// Cast the default values to the appropriate type
	for i, function := range user_config.Functions {
		for j, arg := range function.Args {
			switch arg.Type {
			case "string":
				user_config.Functions[i].Args[j].Default = arg.Default.(string)
			case "int":
				int_default, err := strconv.Atoi(arg.Default.(string))
				if err != nil {
					return nil, fmt.Errorf("error converting default value to int: %v", err)
				}
				user_config.Functions[i].Args[j].Default = int_default
			case "bool":
				bool_default, err := strconv.ParseBool(arg.Default.(string))
				if err != nil {
					return nil, fmt.Errorf("error converting default value to bool: %v", err)
				}
				user_config.Functions[i].Args[j].Default = bool_default
			case "float":
				float_default, err := strconv.ParseFloat(arg.Default.(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error converting default value to float: %v", err)
				}
				user_config.Functions[i].Args[j].Default = float_default
			default:
				return nil, fmt.Errorf("unsupported type: %s", arg.Type)
			}
		}
	}

	return &user_config, nil
}

// Generate the typed YAML file, extracting the function signatures from the source code for validation
func (g *ConfigGenerator) GenerateTypedYAML() error {

	// For each function in the configuration, generate the typed YAML
	for i, function := range g.Config.Functions {
		if function.TargetFunction == "" {
			continue
		}

		var analyzer parsers.LanguageAnalyzer
	
		if !strings.Contains(function.TargetFile, ".") {
			return fmt.Errorf("no file extension found: %s", function.TargetFile)
		}

		// Determine the file extension and use the appropriate analyzer
		file_extensions := strings.Split(function.TargetFile, ".")
		file_extension := file_extensions[len(file_extensions)-1]
		switch file_extension {
		case "py":
			analyzer = &parsers.PythonAnalyzer{}
		default:
			return fmt.Errorf("unsupported file extension: %s", file_extension)
		}

		// Extract the function signature from the source code
		args, err := analyzer.ExtractSignature(function)
		if err != nil {
			return err
		}

		// Update the function with the extracted arguments
		g.Config.Functions[i].Args = args
	}

	// Marshal the config to YAML
	config_yaml, err := yaml.Marshal(g.Config)
	if err != nil {
		return err
	}

	// Write the generated YAML to the file
	err = os.WriteFile(g.ConfigPath, config_yaml, 0644)
	if err != nil {
		return err
	}

	return nil
}