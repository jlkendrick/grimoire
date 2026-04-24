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
	PathToFunction string
	FunctionName   string
	CommandName    string
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

	// If the config is global, parse the individual project configs and set the context type to global
	if config.RegisteredProjects != nil {
		for _, project := range config.RegisteredProjects {
			project_config, err := ParseConfig(project.Path)
			if err != nil {
				return nil, err
			}
			// Store a reference to the spell path that the function originally belongs to
			// Have to do this here so we don't lose what project the function originally belonged to
			for i := range project_config.Functions {
				project_config.Functions[i].ScrollPath = project.Path
			}
			config.Functions = append(config.Functions, project_config.Functions...)
		}

		config.Context = types.ContextTypeGlobal
	
	} else {
		// Need to set the ScrollPaths here as well for clean command to work with local spells
		for i := range config.Functions {
			config.Functions[i].ScrollPath = path
		}
		config.Context = types.ContextTypeLocal
	}

	config.Path = path

	return &config, nil
}

func (g *ConfigGenerator) GenerateFunctionConfig() (types.Function, error) {
	var analyzer parsers.LanguageAnalyzer

	if !strings.Contains(g.PathToFunction, ".") {
		return types.Function{}, fmt.Errorf("no file extension found: %s", g.PathToFunction)
	}

	// Determine the file extension and use the appropriate analyzer
	file_extensions := strings.Split(g.PathToFunction, ".")
	file_extension := file_extensions[len(file_extensions)-1]
	switch file_extension {
	case "py":
		analyzer = &parsers.PythonAnalyzer{}
	case "go":
		analyzer = &parsers.GoAnalyzer{}
	default:
		return types.Function{}, fmt.Errorf("unsupported file extension: %s", file_extension)
	}

	// Extract the function signature from the source code
	args, err := analyzer.ExtractSignature(g.PathToFunction, g.FunctionName)
	if err != nil {
		return types.Function{}, err
	}

	// Cast the default values to the appropriate type
	for i := range args {
		if args[i].Default != nil {
			err := args[i].CastAndSetDefault()
			if err != nil {
				return types.Function{}, err
			}
		}
	}

	name := g.CommandName
	if name == "" {
		name = g.FunctionName
	}
	return types.Function{
		Name:           name,
		TargetFile:     g.PathToFunction,
		TargetFunction: g.FunctionName,
		Args:           args,
	}, nil
}