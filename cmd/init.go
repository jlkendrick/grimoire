package cmd

import (
	"os"
	"fmt"
	"path"

	"github.com/spf13/cobra"
)

var newInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a boilerplate grim.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		// Generate a boilerplate grim.yaml file in the current directory
		boilerplate_yaml := `functions:
		- name: # CLI command associated with running the function 
		  path: # Path to the file containing the function
		  function: # Name of the function to run
		  args:
			- name: # Name of the argument
			  type: # Type of the argument
			  default: # Default value of the argument (optional)
		`
		err := os.WriteFile("grim.yaml", []byte(boilerplate_yaml), 0644)
		if err != nil {
			fmt.Printf("Error writing boilerplate grim.yaml file: %v\n", err)
			return
		}
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}
		fmt.Printf("Boilerplate grim.yaml file generated at %s\n", path.Join(current_dir, "grim.yaml"))
	},
}

func init() {
	rootCmd.AddCommand(newInitCmd)
}