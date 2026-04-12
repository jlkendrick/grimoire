package cmd

import (
	"os"
	"fmt"
	"path"
	"strings"
	"path/filepath"

	"github.com/spf13/cobra"

	config "github.com/jlkendrick/grimoire/config"
	utils "github.com/jlkendrick/grimoire/utils"
)

var add_cmd = &cobra.Command{
	Use:   "add [path_to_function:function_name]",
	Short: "Add a function to the grim.yaml file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !strings.Contains(args[0], ":") {
			fmt.Printf("Error: path_to_function:function_name format is required\n")
			return
		}
		parts := strings.Split(args[0], ":")
		path_to_function := parts[0]
		function_name := parts[1]

		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			fmt.Printf("Error getting global flag: %v\n", err)
			return
		}

		// Determine the path to write the spell to
		var config_path string
		if global {
			config_path, err = utils.ExpandUserPath("~/.grimoire.yaml")
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

		// Make sure the config file exists
		_, err = os.Stat(config_path)
		if os.IsNotExist(err) {
			fmt.Println("Config file does not exist, creating one...")
			err = makeBlankGrimYAMLFile(path.Dir(config_path), true)
			if err != nil {
				fmt.Printf("Error creating config file: %v\n", err)
				return
			}
			return
		}

		// Parse the existing config file
		existing_config, err := config.ParseConfig(config_path)
		if err != nil {
			fmt.Printf("Error parsing config file: %v\n", err)
			return
		}

		// Make the path_to_function relative to the config file that we found
		absolute_path_to_function, err := filepath.Abs(path_to_function)
		if err != nil {
			fmt.Printf("Error getting absolute path to function: %v\n", err)
			return
		}
		path_to_function, err = utils.MakeRelativePath(absolute_path_to_function, filepath.Dir(config_path))
		if err != nil {
			fmt.Printf("Error making path to function relative: %v\n", err)
			return
		}

		config_generator := config.ConfigGenerator{PathToFunction: path_to_function, FunctionName: function_name}
		function_config, err := config_generator.GenerateFunctionConfig()
		if err != nil {
			fmt.Printf("Error generating function config: %v\n", err)
			return
		}
		existing_config.Functions = append(existing_config.Functions, function_config)

		// Write the updated config file
		err = existing_config.Write(config_path)
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}

		fmt.Printf("Function %s added to config file at %s\n", function_name, config_path)
	},
}

func init() {
	add_cmd.Flags().BoolP("global", "g", false, "Add the function to the global grimoire")
	rootCmd.AddCommand(add_cmd)
}