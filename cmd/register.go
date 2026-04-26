package cmd

import (
	"fmt"
	"os"

	core "github.com/jlkendrick/grimoire/core"

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
			scroll_path, found := core.FindLocalScroll(current_dir)
			if !found {
				fmt.Printf("Error: no scroll.yaml file found in the current directory or any parent directories\n")
				return
			}
			path_to_project = scroll_path
		}

		if err := core.RegisterScroll(path_to_project); err != nil {
			fmt.Printf("Error registering scroll: %v\n", err)
			return
		}

		fmt.Printf("%s Bound %s to the global grimoire\n", accent("+"), path_to_project)
	},
}

func init() {
	rootCmd.AddCommand(register_cmd)
}
