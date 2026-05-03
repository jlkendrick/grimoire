/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"fmt"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	cache "github.com/jlkendrick/grimoire/internal/cache"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grimoire",
	Short: "Universal declarative execution framework",
	Long: `Grimoire is a declarative, language-agnostic execution framework that
turns plain functions into fully typed CLI commands using YAML configuration.

Define your functions once in a spell.yaml file, and Grimoire generates
subcommands with proper argument parsing, type coercion, and help text —
no boilerplate required.

  grimoire init          Initialize a spell.yaml in the current directory
  grimoire add <file>    Auto-detect functions and add them to spell.yaml
  grimoire sync          Sync changes from spell.yaml to the CLI
  grimoire <command>     Run any function defined in spell.yaml`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

var staticCommands = map[string]bool{
	"init": true,
	"add": true,
	"sync": true,
	"register": true,
	"clean": true,
	"help": true,
}

func Execute() {
	if err := utils.EnsureGrimoireHome(); err != nil {
		fmt.Printf("Warning: could not initialize grimoire home: %v\n", err)
	}

	// Only build the commands if the user has not requested a static command
	var static_command_called bool
	if len(os.Args) > 1 {
		requested_command := os.Args[1]
		_, ok := staticCommands[requested_command]
		static_command_called = ok
	} else {
		static_command_called = false
	}
	

	if static_command_called {
		// Do nothing
	} else {
		// Load the descriptors and the scroll and cache them for whatever command comes next
		
		descriptors, err := cache.LoadCache()
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		_, err = scroll.LoadScroll("local")
		if err != nil {
			fmt.Printf("Error loading scroll: %v\n", err)
			return
		}

		if descriptors != nil {
			commands, err := GenerateCommands(descriptors)
			if err != nil {
				fmt.Printf("Error generating commands: %v\n", err)
				return
			}
			
			for _, command := range commands {
				rootCmd.AddCommand(command)
			}
		}
	}

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("Error executing root command: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.grimoire.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


