package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	config "github.com/jlkendrick/grimoire/config"
	core "github.com/jlkendrick/grimoire/core"
	types "github.com/jlkendrick/grimoire/types"
	utils "github.com/jlkendrick/grimoire/utils"
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

		command_name, err := cmd.Flags().GetString("name")
		if err != nil {
			fmt.Printf("Error getting name flag: %v\n", err)
			return
		}
		if command_name == "" {
			command_name = function_name
		}

		config_obj, err := resolveAddConfig()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		// Check if a spell with the same command name already exists in the config
		for _, function := range config_obj.Functions {
			if function.Name == command_name {
				fmt.Printf("%s Spell named %s already exists in the scroll\n", accent("+"), spell(command_name))
				return
			}
		}

		absolute_path_to_function, err := filepath.Abs(path_to_function)
		if err != nil {
			fmt.Printf("Error getting absolute path to function: %v\n", err)
			return
		}

		fmt.Printf("%s Divining signature...\n", accent("+"))

		config_generator := config.ConfigGenerator{
			AbsPathToFunction: absolute_path_to_function,
			ScrollPath:        config_obj.Path,
			FunctionName:      function_name,
			CommandName:       command_name,
		}
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

		if err := config_obj.Write(); err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}

		scroll_name := filepath.Base(filepath.Dir(config_obj.Path))
		fmt.Printf("%s Bound to scroll %s\n", accent("+"), spell(scroll_name))
	},
}

// resolveAddConfig returns the local scroll that `add` should write into. If
// no scroll exists in cwd or any parent, initializes one in cwd and registers
// it with the global grimoire.
func resolveAddConfig() (*types.Config, error) {
	current_dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Error getting current directory: %v", err)
	}

	if _, found := core.FindLocalScroll(current_dir); found {
		return core.LoadConfig("local")
	}

	fmt.Printf("%s No scroll found, initializing new scroll\n", accent("+"))
	cfg, err := core.InitScroll(current_dir, false)
	if err != nil {
		return nil, fmt.Errorf("Error initializing scroll: %v", err)
	}
	fmt.Printf("%s Inscribed scroll.yaml\n  · %s\n", accent("+"), dim(cfg.Path))
	if err := core.RegisterScroll(cfg.Path); err != nil {
		return nil, fmt.Errorf("Error registering scroll: %v", err)
	}
	fmt.Printf("%s Bound %s to the global grimoire\n", accent("+"), cfg.Path)
	return cfg, nil
}

func init() {
	add_cmd.Flags().StringP("name", "n", "", "Command name to use for the function")
	rootCmd.AddCommand(add_cmd)
}
