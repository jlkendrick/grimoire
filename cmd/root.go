/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"fmt"

	"github.com/spf13/cobra"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
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
	// "sync": true, TODO
	"register": true,
	"clean": true,
	"weave": true,
	"help": true,
}

func Execute() {
	if err := utils.EnsureGrimoireSetup(); err != nil {
		fmt.Printf("Warning: could not setup grimoire directories and files: %v\n", err)
	}

	// Only build the commands if the user has not requested a static command
	var static_command_called bool
	if len(os.Args) > 1 {
		_, ok := staticCommands[os.Args[1]]
		static_command_called = ok
	}

	if !static_command_called {
		scrolls, err := scroll.LoadScrolls()
		if err != nil {
			fmt.Printf("Error loading scrolls: %v\n", err)
			return
		}

		if err := registerScrollCommands(scrolls); err != nil {
			fmt.Printf("%v\n", err)
			return
		}
	}

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
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
