package cmd

import (
	"fmt"
	"strings"
	"path/filepath"

	"github.com/spf13/cobra"

	config "github.com/jlkendrick/grimoire/config"
	utils "github.com/jlkendrick/grimoire/utils"
	core "github.com/jlkendrick/grimoire/core"
	types "github.com/jlkendrick/grimoire/types"
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

		// Check if a spell with the same command name already exists in the config
		for _, function := range config_obj.Functions {
			if function.Name == command_name {
				fmt.Printf("%s Spell named %s already exists in the scroll\n", accent("+"), spell(command_name))
				return
			}
		}

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

		fmt.Printf("%s Divining signature...\n", accent("+"))

		config_generator := config.ConfigGenerator{PathToFunction: path_to_function, FunctionName: function_name, CommandName: command_name}
		function_config, err := config_generator.GenerateFunctionConfig()
		if err != nil {
			fmt.Printf("Error generating function config: %v\n", err)
			return
		}

		// Format the args tree line
		argParts := make([]string, 0, len(function_config.Args))
		for _, arg := range function_config.Args {
			if arg.Default != nil {
				argParts = append(argParts, fmt.Sprintf("%s:%s=%v", arg.Name, arg.Type, arg.Default))
			} else {
				argParts = append(argParts, fmt.Sprintf("%s:%s", arg.Name, arg.Type))
			}
		}

		// Detect language and dep file for the runtime line
		ext := strings.TrimPrefix(filepath.Ext(absolute_path_to_function), ".")
		lang := ext
		if ext == "py" {
			lang = "python"
		}
		runtimeLine := lang
		depTargets, depFound := utils.UpwardsTraversalForTargets(filepath.Dir(absolute_path_to_function), []string{"pyproject.toml", "requirements.txt"})
		if depFound {
			if _, ok := depTargets["pyproject.toml"]; ok {
				runtimeLine = lang + " · pyproject.toml"
			} else if _, ok := depTargets["requirements.txt"]; ok {
				runtimeLine = lang + " · requirements.txt"
			}
		}

		// Print the signature tree
		fmt.Printf("%s function %s\n", accent("├──"), spell(function_name))
		if len(argParts) > 0 {
			fmt.Printf("%s args %s\n", accent("├──"), strings.Join(argParts, " "))
		}
		fmt.Printf("%s runtime %s\n", accent("└──"), runtimeLine)

		config_obj.Functions = append(config_obj.Functions, function_config)

		// Write the updated config file
		err = config_obj.Write()
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}

		// Scroll name: directory containing the scroll
		scroll_name := filepath.Base(filepath.Dir(config_obj.Path))
		if config_obj.Context == types.ContextTypeGlobal {
			scroll_name = "the grimoire"
		}
		fmt.Printf("%s Bound to scroll %s\n", accent("+"), spell(scroll_name))
	},
}

func init() {
	add_cmd.Flags().BoolP("global", "g", false, "Add the function to the global grimoire")
	add_cmd.Flags().StringP("name", "n", "", "Command name to use for the function")
	rootCmd.AddCommand(add_cmd)
}