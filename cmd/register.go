package cmd

import (
	"os"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	
	utils "github.com/jlkendrick/grimoire/internal/utils"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

var register_cmd = &cobra.Command{
	Use:   "register [path_to_scroll]",
	Short: "Register a project with the global grimoire",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var relative_path_to_scroll string
		var absolute_path_to_scroll string
		var err error

		if len(args) > 0 {
			relative_path_to_scroll = args[0]
			absolute_path_to_scroll, err = filepath.Abs(relative_path_to_scroll)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}

		} else {
			current_dir, err := os.Getwd()
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
			scroll_path, found := scroll.FindLocalScroll(current_dir)
			if !found {
				fmt.Printf("Error: no scroll.yaml file found in the current directory or any parent directories\n")
				return
			}
			// Scroll path is already absolute
			relative_path_to_scroll, err = utils.MakeRelativePath(scroll_path, current_dir)
			if err != nil {
				fmt.Printf("Error making relative path: %v\n", err)
				return
			}
			absolute_path_to_scroll = scroll_path
		}

		if err := scroll.RegisterScroll(absolute_path_to_scroll); err != nil {
			fmt.Printf("Error registering scroll: %v\n", err)
			return
		}

		fmt.Printf("%s Bound %s to the global grimoire\n", accent_style("+"), relative_path_to_scroll)
	},
}

func init() {
	rootCmd.AddCommand(register_cmd)
}
