package cmd

import (
	"fmt"
	"path/filepath"

	config "github.com/jlkendrick/grimoire/config"
	core "github.com/jlkendrick/grimoire/core"
	types "github.com/jlkendrick/grimoire/types"

	"github.com/spf13/cobra"
)

var sync_cmd = &cobra.Command{
	Use:   "sync",
	Short: "Automatically generate arguments for all functions in the scroll.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s Scrying all functions...\n", accent("+"))

		global, err := cmd.Flags().GetBool("global")
		if err != nil {
			fmt.Printf("Error getting global flag: %v\n", err)
			return
		}

		config_type := "local"
		if global {
			config_type = "global"
		}
		config_obj, err := core.LoadConfig(config_type)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		// Refresh args for every function. Each Function carries the path of
		// the scroll it belongs to, so target paths resolve against that scroll
		// even when we loaded the global grimoire (which flattens registered
		// scrolls into config_obj.Functions).
		for i, function := range config_obj.Functions {
			absolute_target_file := function.TargetFile
			if !filepath.IsAbs(absolute_target_file) {
				absolute_target_file = filepath.Join(filepath.Dir(function.ScrollPath), absolute_target_file)
			}
			config_generator := config.ConfigGenerator{
				AbsPathToFunction: absolute_target_file,
				ScrollPath:        function.ScrollPath,
				FunctionName:      function.TargetFunction,
			}
			function_config, err := config_generator.GenerateFunctionConfig()
			if err != nil {
				fmt.Printf("Error generating function config: %v\n", err)
				return
			}
			config_obj.Functions[i].Args = function_config.Args

			prefix := "├──"
			if i == len(config_obj.Functions)-1 {
				prefix = "└──"
			}
			fmt.Printf("%s attuned %s\n", accent(prefix), spell(function.Name))
		}

		if global {
			// Write each registered scroll separately. Calling Write on the
			// global config_obj would inline every project's functions into
			// ~/.grimoire/grimoire.yaml, corrupting the index.
			by_scroll := map[string][]types.Function{}
			for _, fn := range config_obj.Functions {
				by_scroll[fn.ScrollPath] = append(by_scroll[fn.ScrollPath], fn)
			}
			for scroll_path, fns := range by_scroll {
				scroll_cfg := types.Config{Functions: fns, Path: scroll_path}
				if err := scroll_cfg.Write(); err != nil {
					fmt.Printf("Error writing %s: %v\n", scroll_path, err)
					return
				}
			}
		} else {
			if err := config_obj.Write(); err != nil {
				fmt.Printf("Error writing config file: %v\n", err)
				return
			}
		}
		fmt.Printf("%s Harmonized %d spells\n", accent("+"), len(config_obj.Functions))
	},
}

func init() {
	sync_cmd.Flags().BoolP("global", "g", false, "Sync every scroll registered with the global grimoire")
	rootCmd.AddCommand(sync_cmd)
}
