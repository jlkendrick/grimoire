package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	extract "github.com/jlkendrick/grimoire/internal/extract"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

func buildFunctionCommand(function_descriptor descriptor.FunctionDescriptor, scroll_path string) (*cobra.Command, error) {
	command := &cobra.Command{
		Use: function_descriptor.CommandName,
		Run: func(cmd *cobra.Command, args []string) {
			var resolved_descriptor descriptor.FunctionDescriptor
			// Pre-run: check if the function descriptor is stale relative to the source code.
			// Scroll hash is checked in the root command before generating commands.
			source_hash, err := utils.HashFile(function_descriptor.AbsPathToSourceFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error hashing source file: %v\n", err)
				os.Exit(1)
			}

			if source_hash != function_descriptor.SourceHash {
				fmt.Printf("%s Source code change detected. Updating casting recipe...\n", utils.SpellStyle("+"))
				function_descriptor_generator := extract.FunctionDescriptorGenerator{
					CommandName:         function_descriptor.CommandName,
					FunctionName:        function_descriptor.FunctionName,
					RelPathToSourceFile: function_descriptor.RelPathToSourceFile,
					AbsPathToSourceFile: function_descriptor.AbsPathToSourceFile,
					ScrollPath:          scroll_path,
					SpellHash:           function_descriptor.SpellHash, // Is fresh since root command checks for staleness
					Interpreter:         function_descriptor.Interpreter,
				}
				resolved_descriptor, err = function_descriptor_generator.Generate()
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error generating spell descriptor: %v\n", err)
					os.Exit(1)
				}

				if err := cache.AddFunctionDescriptor(resolved_descriptor); err != nil {
					fmt.Fprintf(os.Stderr, "Error caching function descriptor: %v\n", err)
					os.Exit(1)
				}
			} else {
				resolved_descriptor = function_descriptor
			}

			payload := buildPayload(resolved_descriptor, cmd)

			start := time.Now()
			runResult, err := runtime.Run(&resolved_descriptor, payload)
			elapsed := time.Since(start)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error executing function: %v\n", err)
				os.Exit(1)
			}

			fmt.Println(string(runResult.Output))

			footerParts := []string{fmt.Sprintf("%.2fs", elapsed.Seconds())}
			if runResult.CacheStatus != "" {
				footerParts = append(footerParts, runResult.CacheStatus)
			}
			if runResult.Runtime != "" {
				footerParts = append(footerParts, runResult.Runtime)
			}
			fmt.Fprintf(os.Stderr, "\n%s %s\n", accent_style("◈"), dim_style(strings.Join(footerParts, " · ")))
		},
	}

	for _, param := range function_descriptor.Params {
		if err := registerParamFlag(command, param); err != nil {
			return nil, err
		}
	}

	return command, nil
}
