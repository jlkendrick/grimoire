package cmd

import (
	"fmt"
	"strings"
	"path/filepath"

	"github.com/spf13/cobra"

	config "github.com/jlkendrick/grimoire/config"
	utils "github.com/jlkendrick/grimoire/utils"
	core "github.com/jlkendrick/grimoire/core"
)

var add_cmd = &cobra.Command{
	Use:   "add [path_to_function:function_name]",
	Short: "Add a function to the scroll.yaml file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !strings.Contains(args[0], ":") {
			fmt.Printf("Error: path_to_function:function_name format is required\n")
			return
		}
		parts := strings.Split(args[0], ":")
		path_to_function := parts[0]
		function_name := parts[1]

		// Get the global flag
		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			fmt.Printf("Error getting global flag: %v\n", err)
			return
		}

		// Get the name flag
		command_name, err := cmd.Flags().GetString("name")
		if err != nil {
			fmt.Printf("Error getting name flag: %v\n", err)
			return
		}
		if command_name == "" {
			command_name = function_name
		}

		var config_type string
		if global {
			config_type = "global"
		} else {
			config_type = "local"
		}
		config_obj, err := core.LoadConfig(config_type)

		// Make the path_to_function relative to the config file that we found
		absolute_path_to_function, err := filepath.Abs(path_to_function)
		if err != nil {
			fmt.Printf("Error getting absolute path to function: %v\n", err)
			return
		}
		path_to_function, err = utils.MakeRelativePath(absolute_path_to_function, filepath.Dir(config_obj.Path))
		if err != nil {
			fmt.Printf("Error making path to function relative: %v\n", err)
			return
		}

		config_generator := config.ConfigGenerator{PathToFunction: path_to_function, FunctionName: function_name, CommandName: command_name}
		function_config, err := config_generator.GenerateFunctionConfig()
		if err != nil {
			fmt.Printf("Error generating function config: %v\n", err)
			return
		}
		config_obj.Functions = append(config_obj.Functions, function_config)

		// Write the updated config file
		err = config_obj.Write()
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}

		fmt.Printf("Function %s added to config file at %s\n", command_name, config_obj.Path)
	},
}

func init() {
	add_cmd.Flags().BoolP("global", "g", false, "Add the function to the global grimoire")
	add_cmd.Flags().StringP("name", "n", "", "Command name to use for the function")
	rootCmd.AddCommand(add_cmd)
}