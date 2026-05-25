package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// GenerateCommands builds cobra commands from a single scroll's descriptor
// cache. resolvedNames maps a descriptor's CommandName to its final cobra
// Use string (which may be a bare command name or a `<scroll>.<command>`
// dot-form). scrollPath is appended to dot-form commands' Short so
// `grimoire help` is navigable across scrolls.
func GenerateCommands(descriptor_cache *cache.DescriptorCache, resolvedNames map[string]string, scrollPath string) ([]*cobra.Command, error) {
	commands := []*cobra.Command{}

	function_descriptors := make([]descriptor.FunctionDescriptor, 0, len(descriptor_cache.Functions))
	for _, fd := range descriptor_cache.Functions {
		function_descriptors = append(function_descriptors, fd)
	}
	sort.Slice(function_descriptors, func(i, j int) bool {
		return function_descriptors[i].CommandName < function_descriptors[j].CommandName
	})
	for _, fd := range function_descriptors {
		command, err := buildFunctionCommand(fd)
		if err != nil {
			return nil, err
		}
		applyResolvedName(command, fd.CommandName, resolvedNames, scrollPath)
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
		applyResolvedName(command, pd.CommandName, resolvedNames, scrollPath)
		commands = append(commands, command)
	}

	return commands, nil
}

func applyResolvedName(cmd *cobra.Command, descriptorName string, resolvedNames map[string]string, scrollPath string) {
	resolved, ok := resolvedNames[descriptorName]
	if !ok || resolved == "" {
		return
	}
	cmd.Use = resolved
	if strings.Contains(resolved, ".") && cmd.Short == "" {
		cmd.Short = fmt.Sprintf("from %s", scrollPath)
	}
}
