package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// registerParamFlag adds a single typed flag to command from a ParamDescriptor.
// Defaults are stored as strings in the descriptor IR (set by extractors and
// the YAML scroll loader); cast to the param's declared type here when
// constructing the cobra flag.
func registerParamFlag(command *cobra.Command, param descriptor.ParamDescriptor) error {
	defaultStr, hasDefault, err := stringDefault(param)
	if err != nil {
		return err
	}

	switch param.ResolvedType.Name {
	case "string", "str":
		if !hasDefault {
			command.Flags().StringP(param.Name, "", "", "")
			command.MarkFlagRequired(param.Name)
			return nil
		}
		command.Flags().StringP(param.Name, "", defaultStr, "")

	case "integer", "int":
		if !hasDefault {
			command.Flags().IntP(param.Name, "", 0, "")
			command.MarkFlagRequired(param.Name)
			return nil
		}
		def, err := strconv.Atoi(defaultStr)
		if err != nil {
			return fmt.Errorf("default value for %s is not an int: %v", param.Name, err)
		}
		command.Flags().IntP(param.Name, "", def, "")

	case "boolean", "bool":
		if !hasDefault {
			command.Flags().BoolP(param.Name, "", false, "")
			command.MarkFlagRequired(param.Name)
			return nil
		}
		def, err := strconv.ParseBool(defaultStr)
		if err != nil {
			return fmt.Errorf("default value for %s is not a bool: %v", param.Name, err)
		}
		command.Flags().BoolP(param.Name, "", def, "")

	case "float":
		if !hasDefault {
			command.Flags().Float64P(param.Name, "", 0.0, "")
			command.MarkFlagRequired(param.Name)
			return nil
		}
		def, err := strconv.ParseFloat(defaultStr, 64)
		if err != nil {
			return fmt.Errorf("default value for %s is not a float: %v", param.Name, err)
		}
		command.Flags().Float64P(param.Name, "", def, "")

	default:
		return fmt.Errorf("unsupported type: %s", param.ResolvedType.Name)
	}

	return nil
}

func buildPayload(function_descriptor descriptor.FunctionDescriptor, cmd *cobra.Command) map[string]interface{} {
	payload := make(map[string]interface{})

	// Type strings here must match the canonical set handled by registerParamFlag.
	for _, param := range function_descriptor.Params {
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
// descriptor IR stores defaults as strings, but JSON cache round-trips can
// preserve historic non-string values, so be defensive.
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
