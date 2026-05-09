package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	utils "github.com/jlkendrick/grimoire/internal/utils"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	extract "github.com/jlkendrick/grimoire/internal/extract"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
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

		// 1. Find or create scroll to add function to
		scroll_obj, err := resolveAddScroll()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		// // Check if a spell with the same command name already exists in the config
		// for _, spell := range scroll_obj.Spells {
		// 	if spell.Command == command_name {
		// 		fmt.Printf("%s Spell named %s already exists in the scroll\n", accent_style("+"), spell_style(command_name))
		// 		return
		// 	}
		// }

		absolute_path_to_function, err := filepath.Abs(path_to_function)
		if err != nil {
			fmt.Printf("Error getting absolute path to function: %v\n", err)
			return
		}

		// 2. Extract -> function descriptor (partially complete)
		fmt.Printf("%s Divining signature...\n", accent_style("+"))
		rel_path_to_function, err := utils.MakeRelativePath(absolute_path_to_function, filepath.Dir(scroll_obj.Path))
		if err != nil {
			fmt.Printf("Error making relative path: %v\n", err)
			return
		}
		function_descriptor_generator := extract.FunctionDescriptorGenerator{
			CommandName: command_name,
			FunctionName: function_name,
			AbsPathToSourceFile: absolute_path_to_function,
			RelPathToSourceFile: rel_path_to_function,
			ScrollPath: scroll_obj.Path,
		}
		// Generate() fills in the above fields, SOURCE HASH ONLY, and params
		// Spell hash is set later when we actually create the spell
		function_descriptor, err := function_descriptor_generator.Generate()
		if err != nil {
			fmt.Printf("Error generating spell descriptor: %v\n", err)
			return
		}

		// // Resolve -> function descriptor (fully resolved)
		// err = resolve.ResolveFunctionDescriptor(&function_descriptor)
		// if err != nil {
		// 	fmt.Printf("Error resolving function descriptor: %v\n", err)
		// 	return
		// }

		// 3. Read existing scroll entry to perform overrides if spell is already defined
		var existing_spell scroll.Spell
		for _, spell := range scroll_obj.Spells {
			if spell.Command == command_name {
				existing_spell = spell
				break
			}
		}
		if existing_spell.Command != "" {
			err = resolve.MergeSpellIntoFunctionDescriptor(existing_spell, &function_descriptor)
			if err != nil {
				fmt.Printf("Error merging spell into function descriptor: %v\n", err)
				return
			}
		}
		
		// 4. Write the minimal entry to scroll.yaml (unless overrides exist)
		var spell scroll.Spell
		// If there was an existing spell, write it as is
		if existing_spell.Command != "" {
			// Do nothing
			spell = existing_spell
		// If not, generate a minimal spell from the function descriptor
		} else {
			spell, err = scroll.GenerateMinimalSpellFromFunctionDescriptor(function_descriptor)
			if err != nil {
				fmt.Printf("Error generating minimal spell: %v\n", err)
				return
			}
			scroll_obj.Spells = append(scroll_obj.Spells, spell)
		}
		if err := scroll_obj.Write(); err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}

		// Set the spell hash for the function descriptor
		spell_hash, err := spell.Hash()
		if err != nil {
			fmt.Printf("Error hashing spell: %v\n", err)
			return
		}
		function_descriptor.SpellHash = spell_hash
		
		// 5. Write the final descriptor to the cache
		err = cache.AddFunctionDescriptor(function_descriptor)
		if err != nil {
			fmt.Printf("Error writing cache: %v\n", err)
			return
		}

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


		scroll_name := filepath.Base(filepath.Dir(scroll_obj.Path))
		fmt.Printf("%s Bound to scroll %s\n", accent_style("+"), spell_style(scroll_name))
	},
}

// resolveAddScroll returns the local scroll that `add` should write into. If
// no scroll exists in cwd or any parent, initializes one in cwd and registers
// it with the global grimoire.
func resolveAddScroll() (*scroll.Scroll, error) {
	if local, found, err := scroll.LoadLocalScroll(); err != nil {
		return nil, fmt.Errorf("Error loading local scroll: %v", err)
	} else if found {
		return local, nil
	}

	current_dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Error getting current directory: %v", err)
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
