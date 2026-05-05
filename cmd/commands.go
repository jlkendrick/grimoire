package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	cache "github.com/jlkendrick/grimoire/internal/cache"
	runtime "github.com/jlkendrick/grimoire/internal/runtime"
	extract "github.com/jlkendrick/grimoire/internal/extract"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func GenerateCommands(descriptor_cache *cache.DescriptorCache) ([]*cobra.Command, error) {
	commands := []*cobra.Command{}

	for _, function_descriptor := range descriptor_cache.Functions {
		command := &cobra.Command{
			Use: function_descriptor.CommandName,
			Run: func(cmd *cobra.Command, args []string) {
				var resolved_descriptor descriptor.FunctionDescriptor
				// Pre-run: check if the function descriptor is stale relative to the source code
				// Scroll hash is checked in the root command before generating commands
				source_hash, err := utils.HashFile(function_descriptor.AbsPathToSourceFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error hashing source file: %v\n", err)
					os.Exit(1)
				}

				if source_hash != function_descriptor.SourceHash {
					fmt.Printf("%s Source code change detected. Updating casting recipe...\n", utils.SpellStyle("+"))
					// Re-extract the function descriptor
					function_descriptor_generator := extract.FunctionDescriptorGenerator{
						CommandName: function_descriptor.CommandName,
						FunctionName: function_descriptor.FunctionName,
						RelPathToSourceFile: function_descriptor.RelPathToSourceFile,
						AbsPathToSourceFile: function_descriptor.AbsPathToSourceFile,
						ScrollPath: descriptor_cache.ScrollPath,
						SpellHash: function_descriptor.SpellHash, // Is fresh since root command checks for staleness
						Interpreter: function_descriptor.Interpreter,
					}
					resolved_descriptor, err = function_descriptor_generator.Generate()
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error generating spell descriptor: %v\n", err)
						os.Exit(1)
					}

					// Cache the resolved descriptor
					err = cache.AddFunctionDescriptor(resolved_descriptor)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error caching function descriptor: %v\n", err)
						os.Exit(1)
					}
					
				} else {
					// Use the cached descriptor
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
			// Defaults are stored as strings in the descriptor IR; cast to the
			// param's declared type here when constructing the cobra flag.
			defaultStr, hasDefault, err := stringDefault(param)
			if err != nil {
				return nil, err
			}

			switch param.ResolvedType.Name {
			case "string", "str":
				if !hasDefault {
					command.Flags().StringP(param.Name, "", "", "")
					command.MarkFlagRequired(param.Name)
					continue
				}
				command.Flags().StringP(param.Name, "", defaultStr, "")

			case "integer", "int":
				if !hasDefault {
					command.Flags().IntP(param.Name, "", 0, "")
					command.MarkFlagRequired(param.Name)
					continue
				}
				def, err := strconv.Atoi(defaultStr)
				if err != nil {
					return nil, fmt.Errorf("default value for %s is not an int: %v", param.Name, err)
				}
				command.Flags().IntP(param.Name, "", def, "")

			case "boolean", "bool":
				if !hasDefault {
					command.Flags().BoolP(param.Name, "", false, "")
					command.MarkFlagRequired(param.Name)
					continue
				}
				def, err := strconv.ParseBool(defaultStr)
				if err != nil {
					return nil, fmt.Errorf("default value for %s is not a bool: %v", param.Name, err)
				}
				command.Flags().BoolP(param.Name, "", def, "")

			case "float":
				if !hasDefault {
					command.Flags().Float64P(param.Name, "", 0.0, "")
					command.MarkFlagRequired(param.Name)
					continue
				}
				def, err := strconv.ParseFloat(defaultStr, 64)
				if err != nil {
					return nil, fmt.Errorf("default value for %s is not a float: %v", param.Name, err)
				}
				command.Flags().Float64P(param.Name, "", def, "")

			default:
				return nil, fmt.Errorf("unsupported type: %s", param.ResolvedType.Name)
			}
		}

		commands = append(commands, command)
	}

	return commands, nil
}

func buildPayload(function_descriptor descriptor.FunctionDescriptor, cmd *cobra.Command) map[string]interface{} {
	// Initialize the dynamic payload map
	payload := make(map[string]interface{})

	// Loop through the YAML params for this specific function
	for _, param := range function_descriptor.Params {
		// Must match the type strings used in GenerateCommands (including aliases like str/int/bool).
		switch param.ResolvedType.Name {
		case "integer", "int":
			val, _ := cmd.Flags().GetInt(param.Name)
			payload[param.Name] = val
		case "string", "str":
			val, _ := cmd.Flags().GetString(param.Name)
			payload[param.Name] = val
		case "boolean", "bool":
			val, _ := cmd.Flags().GetBool(param.Name)
			payload[param.Name] = val
		case "float":
			val, _ := cmd.Flags().GetFloat64(param.Name)
			payload[param.Name] = val
		}
	}

	return payload
}

// stringDefault extracts a string Default from a ParamDescriptor. The
// descriptor IR stores defaults as strings (set by extractors and the YAML
// scroll loader), but JSON cache round-trips can preserve historic non-string
// values, so be defensive.
func stringDefault(param descriptor.ParamDescriptor) (string, bool, error) {
	if param.Default == nil {
		return "", false, nil
	}
	switch v := param.Default.(type) {
	case string:
		return v, true, nil
	default:
		return "", false, fmt.Errorf("default value for %s must be a string in the descriptor IR, got %T", param.Name, param.Default)
	}
}