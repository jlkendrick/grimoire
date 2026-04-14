/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"fmt"

	core "github.com/jlkendrick/grimoire/core"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "grimoire",
	Short: "Universal declarative execution framework",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		// Build the commands
		config, err := core.LoadConfig("local")
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		if config != nil {
			commands, err := GenerateCommands(config)
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


