/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"
	"fmt"
	"path"

	utils "github.com/jlkendrick/grimoire/utils"
	config "github.com/jlkendrick/grimoire/config"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sigil",
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	current_dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}
	var config_path string
	matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"grim.yaml"})
	if found {
		config_path = matched_targets["grim.yaml"]
	} else {
		config_path = path.Join(current_dir, "grim.yaml")
	}
	config, err := config.ParseConfig(config_path)

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
	

	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sigil.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


