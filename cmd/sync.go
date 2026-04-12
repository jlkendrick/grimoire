package cmd

import (
	"os"
	"fmt"
	"path"

	utils "github.com/jlkendrick/grimoire/utils"
	config "github.com/jlkendrick/grimoire/config"

	"github.com/spf13/cobra"
)

var sync_cmd = &cobra.Command{
	Use:   "sync",
	Short: "Automatically generate arguments for all functions in the grim.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Automatically generating arguments for all functions in the grim.yaml file...")

		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			fmt.Printf("Error getting global flag: %v\n", err)
			return
		}

		// Determine the path to write the spell to
		var config_path string
		if global {
			// UPDATE LATER WITH PERMANENT CONFIG FILE PATH
			config_path, err = utils.ExpandUserPath("~/Code/Projects/grimoire/grim.yaml")
			if err != nil {
				fmt.Printf("Error expanding user path: %v\n", err)
				return
			}
		} else {
			current_dir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error getting current directory: %v\n", err)
				return
			}
			matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"grim.yaml"})
			if found {
				config_path = matched_targets["grim.yaml"]
			} else {
				config_path = path.Join(current_dir, "grim.yaml")
			}
		}

		// Parse the existing config file
		existing_config, err := config.ParseConfig(config_path)
		if err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
			return
		}

		// For each function in the config, generate the function config
		for i, function := range existing_config.Functions {
			config_generator := config.ConfigGenerator{PathToFunction: function.TargetFile, FunctionName: function.TargetFunction}
			function_config, err := config_generator.GenerateFunctionConfig()
			if err != nil {
				fmt.Printf("Error generating function config: %v\n", err)
				return
			}
			existing_config.Functions[i].Args = function_config.Args
		}

		// Write the updated config file
		err = existing_config.Write(config_path)
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}
		fmt.Printf("Config file updated at %s\n", config_path)
	},
}

func init() {
	sync_cmd.Flags().BoolP("global", "g", false, "Sync the global grimoire")
	rootCmd.AddCommand(sync_cmd)
}