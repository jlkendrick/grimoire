package cmd

import (
	"os"
	"fmt"
	"path"

	"github.com/spf13/cobra"
)

func makeBlankGrimYAMLFile(directory string, include_boilerplate bool) error {
	var yaml_content string
	if include_boilerplate {
		yaml_content = `functions:
			- name: # CLI command associated with running the function 
			  path: # Path to the file containing the function
			  function: # Name of the function to run
			  args:
				- name: # Name of the argument
				  type: # Type of the argument
				  default: # Default value of the argument (optional)
		`
	} else {
		yaml_content = "functions:\n"
	}
	err := os.WriteFile(path.Join(directory, "grim.yaml"), []byte(yaml_content), 0644)
	if err != nil {
		return fmt.Errorf("error writing boilerplate grim.yaml file: %v", err)
	}
	return nil
}

var newInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a boilerplate grim.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		// Generate a boilerplate grim.yaml file in the current directory
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}
		err = makeBlankGrimYAMLFile(current_dir, true)
		if err != nil {
			fmt.Printf("Error generating boilerplate grim.yaml file: %v\n", err)
			return
		}
		fmt.Printf("Boilerplate grim.yaml file generated at %s\n", path.Join(current_dir, "grim.yaml"))
	},
}

func init() {
	rootCmd.AddCommand(newInitCmd)
}