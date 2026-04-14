package cmd

import (
	"os"
	"fmt"

	core "github.com/jlkendrick/grimoire/core"
	utils "github.com/jlkendrick/grimoire/utils"
	types "github.com/jlkendrick/grimoire/types"

	"github.com/spf13/cobra"
)

var register_cmd = &cobra.Command{
	Use:   "register [path_to_project]",
	Short: "Register a project with the global grimoire",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var path_to_project string
		if len(args) > 0 {
			path_to_project = args[0]
		} else {
			current_dir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			matched_targets, found := utils.UpwardsTraversalForTargets(current_dir, []string{"grim.yaml"})
			if found {
				path_to_project = matched_targets["grim.yaml"]
			} else {
				fmt.Printf("Error: no grim.yaml file found in the current directory or any parent directories\n")
				return
			}
		}

		config, err := core.LoadConfig("global")
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			return
		}

		config.RegisteredProjects = append(config.RegisteredProjects, types.Project{Path: path_to_project})

		err = config.Write()
		if err != nil {
			fmt.Printf("Error writing config: %v\n", err)
			return
		}

		fmt.Printf("Project %s registered with the global grimoire\n", path_to_project)
	},
}

func init() {
	rootCmd.AddCommand(register_cmd)
}