package cmd

import (
	"fmt"
	"path/filepath"
	
	config "github.com/jlkendrick/grimoire/config"
	core "github.com/jlkendrick/grimoire/core"

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

		var config_type string
		if global {
			config_type = "global"
		} else {
			config_type = "local"
		}
		config_obj, err := core.LoadConfig(config_type)

		// For each function in the config, generate the function config
		for i, function := range config_obj.Functions {
			absolute_target_file := filepath.Join(filepath.Dir(config_obj.Path), function.TargetFile)
			config_generator := config.ConfigGenerator{AbsPathToFunction: absolute_target_file, ScrollPath: config_obj.Path, FunctionName: function.TargetFunction}
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

		// Write the updated config file
		err = config_obj.Write()
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}
		fmt.Printf("%s Harmonized %d spells\n", accent("+"), len(config_obj.Functions))
	},
}

func init() {
	sync_cmd.Flags().BoolP("global", "g", false, "Sync the global grimoire")
	rootCmd.AddCommand(sync_cmd)
}