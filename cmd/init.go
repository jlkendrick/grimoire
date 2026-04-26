package cmd

import (
	"errors"
	"fmt"
	"os"

	core "github.com/jlkendrick/grimoire/core"

	"github.com/spf13/cobra"
)

var init_cmd = &cobra.Command{
	Use:   "init",
	Short: "Create a boilerplate scroll.yaml file",
	Run: func(cmd *cobra.Command, args []string) {
		current_dir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}

		cfg, err := core.InitScroll(current_dir, true)
		if errors.Is(err, core.ErrScrollExists) {
			fmt.Printf("A scroll already exists at %s\n", dim(current_dir+"/scroll.yaml"))
			return
		}
		if err != nil {
			fmt.Printf("Error generating boilerplate scroll.yaml file: %v\n", err)
			return
		}
		fmt.Printf("%s Inscribed scroll.yaml\n  · %s\n", accent("+"), dim(cfg.Path))
	},
}

func init() {
	rootCmd.AddCommand(init_cmd)
}
