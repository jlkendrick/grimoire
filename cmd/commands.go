package cmd

import (
	"sort"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func GenerateCommands(descriptor_cache *cache.DescriptorCache) ([]*cobra.Command, error) {
	commands := []*cobra.Command{}

	function_descriptors := make([]descriptor.FunctionDescriptor, 0, len(descriptor_cache.Functions))
	for _, fd := range descriptor_cache.Functions {
		function_descriptors = append(function_descriptors, fd)
	}
	sort.Slice(function_descriptors, func(i, j int) bool {
		return function_descriptors[i].CommandName < function_descriptors[j].CommandName
	})
	for _, fd := range function_descriptors {
		command, err := buildFunctionCommand(fd, descriptor_cache.ScrollPath)
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}

	pipeline_descriptors := make([]descriptor.PipelineDescriptor, 0, len(descriptor_cache.Pipelines))
	for _, pd := range descriptor_cache.Pipelines {
		pipeline_descriptors = append(pipeline_descriptors, pd)
	}
	sort.Slice(pipeline_descriptors, func(i, j int) bool {
		return pipeline_descriptors[i].CommandName < pipeline_descriptors[j].CommandName
	})
	for _, pd := range pipeline_descriptors {
		command, err := buildPipelineCommand(pd, descriptor_cache)
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}

	return commands, nil
}
