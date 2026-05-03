package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	extract "github.com/jlkendrick/grimoire/internal/extract"
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
		for _, spell := range config_obj.Spells {
			if spell.Command == command_name {
				fmt.Printf("%s Spell named %s already exists in the scroll\n", accent_style("+"), spell_style(command_name))
				return
			}
		}

		absolute_path_to_function, err := filepath.Abs(path_to_function)
		if err != nil {
			fmt.Printf("Error getting absolute path to function: %v\n", err)
			return
		}

		fmt.Printf("%s Divining signature...\n", accent_style("+"))

		// Generate the function descriptor
		function_descriptor_generator := extract.FunctionDescriptorGenerator{
			AbsPathToFunction: absolute_path_to_function,
			FunctionName:   	 function_name,
		}
		function_descriptor, err := function_descriptor_generator.GenerateDescriptor()
		if err != nil {
			fmt.Printf("Error generating spell descriptor: %v\n", err)
			return
		}
		// Manually set the command name and scroll path (not needed by the extractor above)
		function_descriptor.CommandName = command_name
		function_descriptor.ScrollPath = config_obj.Path

		// Format the args tree line
		argParts := make([]string, 0, len(function_descriptor.Params))
		for _, param := range function_descriptor.Params {
			if param.Default != nil {
				argParts = append(argParts, fmt.Sprintf("%s:%s=%v", param.Name, param.ResolvedType.Name, param.Default))
			} else {
				argParts = append(argParts, fmt.Sprintf("%s:%s", param.Name, param.ResolvedType.Name))
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
		fmt.Printf("%s function %s\n", accent_style("├──"), spell_style(function_name))
		if len(argParts) > 0 {
			fmt.Printf("%s args %s\n", accent_style("├──"), strings.Join(argParts, " "))
		}
		fmt.Printf("%s runtime %s\n", accent_style("└──"), runtimeLine)

		// Minify the spell descriptor to just include the command, path, and function name
		config_obj.Spells = append(config_obj.Spells, scroll.GenerateSpellFromFunctionDescriptor(function_descriptor))

		if err := config_obj.Write(); err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}

		scroll_name := filepath.Base(filepath.Dir(config_obj.Path))
		fmt.Printf("%s Bound to scroll %s\n", accent_style("+"), spell_style(scroll_name))
	},
}

// resolveAddConfig returns the local scroll that `add` should write into. If
// no scroll exists in cwd or any parent, initializes one in cwd and registers
// it with the global grimoire.
func resolveAddConfig() (*scroll.Scroll, error) {
	current_dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Error getting current directory: %v", err)
	}

	if _, found := scroll.FindLocalScroll(current_dir); found {
		return scroll.LoadScroll("local")
	}

	fmt.Printf("%s No scroll found, initializing new scroll\n", accent_style("+"))
	cfg, err := scroll.InitScroll(current_dir, false)
	if err != nil {
		return nil, fmt.Errorf("Error initializing scroll: %v", err)
	}
	fmt.Printf("%s Inscribed scroll.yaml\n  · %s\n", accent_style("+"), dim_style(cfg.Path))
	if err := scroll.RegisterScroll(cfg.Path); err != nil {
		return nil, fmt.Errorf("Error registering scroll: %v", err)
	}
	fmt.Printf("%s Bound %s to the global grimoire\n", accent_style("+"), cfg.Path)
	return cfg, nil
}

func init() {
	add_cmd.Flags().StringP("name", "n", "", "Command name to use for the function")
	rootCmd.AddCommand(add_cmd)
}
