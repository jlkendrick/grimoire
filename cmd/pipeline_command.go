package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func buildPipelineCommand(pipeline_descriptor descriptor.PipelineDescriptor) (*cobra.Command, error) {
	command := &cobra.Command{
		Use: pipeline_descriptor.CommandName,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Running pipeline:", pipeline_descriptor.CommandName)
		},
	}
	return command, nil
}
