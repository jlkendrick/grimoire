package cmd

import (
	"fmt"
	
	config "github.com/jlkendrick/grimoire/config"
	core "github.com/jlkendrick/grimoire/core"

	"github.com/spf13/cobra"
)

var sync_cmd = &cobra.Command{
	Use:   "sync",
	Short: "Automatically generate arguments for all functions in the scroll.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("+ Scrying all functions...")

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
			config_generator := config.ConfigGenerator{PathToFunction: function.TargetFile, FunctionName: function.TargetFunction}
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
			fmt.Printf("%s attuned %s\n", prefix, function.Name)
		}

		// Write the updated config file
		err = config_obj.Write()
		if err != nil {
			fmt.Printf("Error writing config file: %v\n", err)
			return
		}
		fmt.Printf("+ Harmonized %d spells\n", len(config_obj.Functions))
	},
}

func init() {
	sync_cmd.Flags().BoolP("global", "g", false, "Sync the global grimoire")
	rootCmd.AddCommand(sync_cmd)
}