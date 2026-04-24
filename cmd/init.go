package cmd

import (
	"fmt"
	"os"
	"path"

	types "github.com/jlkendrick/grimoire/types"

	"github.com/goccy/go-yaml"
	"github.com/spf13/cobra"
)

func makeBlankGrimYAMLFile(directory string, include_boilerplate bool) error {
	cfg := types.Config{}
	opts := []yaml.EncodeOption{yaml.Indent(2), yaml.IndentSequence(true)}

	if include_boilerplate {
		cfg.Functions = []types.Function{
			{
				Name:           "hello_world",
				TargetFile:     "path/to/hello_world.py",
				TargetFunction: "hello_world",
				Args: []types.Arg{
					{Name: "n", Type: "int", Default: 1},
				},
			},
		}
		opts = append(opts, yaml.WithComment(yaml.CommentMap{
			"$.functions[0].name":     []*yaml.Comment{yaml.LineComment("CLI command associated with running the function")},
			"$.functions[0].path":     []*yaml.Comment{yaml.LineComment("Path to the file containing the function")},
			"$.functions[0].function": []*yaml.Comment{yaml.LineComment("Name of the function to run")},
			"$.functions[0].args[0].name":     []*yaml.Comment{yaml.LineComment("Name of the argument")},
			"$.functions[0].args[0].type":     []*yaml.Comment{yaml.LineComment("Type of the argument")},
			"$.functions[0].args[0].default": []*yaml.Comment{yaml.LineComment("Default value of the argument (optional)")},
		}))
	}

	out, err := yaml.MarshalWithOptions(&cfg, opts...)
	if err != nil {
		return fmt.Errorf("error marshaling scroll.yaml: %w", err)
	}
	err = os.WriteFile(path.Join(directory, "scroll.yaml"), out, 0644)
	if err != nil {
		return fmt.Errorf("error writing boilerplate scroll.yaml file: %v", err)
	}
	return nil
}

var init_cmd = &cobra.Command{
	Use:   "init",
	Short: "Create a boilerplate scroll.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		// Generate a boilerplate scroll.yaml file in the current directory
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}

		// Check if a scroll.yaml file already exists
		if _, err := os.Stat(path.Join(current_dir, "scroll.yaml")); !os.IsNotExist(err) {
			fmt.Printf("A scroll already exists at %s\n", dim(path.Join(current_dir, "scroll.yaml")))
			return
		}

		err = makeBlankGrimYAMLFile(current_dir, true)
		if err != nil {
			fmt.Printf("Error generating boilerplate scroll.yaml file: %v\n", err)
			return
		}
		fmt.Printf("%s Inscribed scroll.yaml\n  · %s\n", accent("+"), dim(path.Join(current_dir, "scroll.yaml")))
	},
}

func init() {
	rootCmd.AddCommand(init_cmd)
}