package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// registerParamFlag adds a single typed flag to command from a ParamDescriptor.
// param.Default is the typed Go value of the parsed default literal. The
// asInt/asFloat/asBool/asString helpers absorb the type variance introduced
// by the JSON cache round-trip (numbers decoded into float64) and goccy
// go-yaml (which may produce uint64 for unsigned ints).
func registerParamFlag(command *cobra.Command, param descriptor.ParamDescriptor) error {
	hasDefault := param.Default != nil

	switch param.ResolvedType.Name {
	case "string", "str":
		if !hasDefault {
			command.Flags().StringP(param.Name, "", "", "")
			command.MarkFlagRequired(param.Name)
			return nil
		}
		def, err := asString(param.Default)
		if err != nil {
			return fmt.Errorf("default value for %s is not a string: %v", param.Name, err)
		}
		command.Flags().StringP(param.Name, "", def, "")

	case "integer", "int":
		if !hasDefault {
			command.Flags().IntP(param.Name, "", 0, "")
			command.MarkFlagRequired(param.Name)
			return nil
		}
		def, err := asInt(param.Default)
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
		def, err := asBool(param.Default)
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
		def, err := asFloat(param.Default)
		if err != nil {
			return fmt.Errorf("default value for %s is not a float: %v", param.Name, err)
		}
		command.Flags().Float64P(param.Name, "", def, "")

	default:
		return fmt.Errorf("unsupported type: %s", param.ResolvedType.Name)
	}

	return nil
}

func asString(v any) (string, error) {
	if s, ok := v.(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("expected string, got %T", v)
}

func asInt(v any) (int, error) {
	switch x := v.(type) {
	case int:
		return x, nil
	case int64:
		return int(x), nil
	case uint64:
		return int(x), nil
	case float64:
		// JSON cache round-trip collapses int64 into float64. Accept whole
		// numbers; reject anything fractional since that signals a real type
		// mismatch upstream.
		if x != float64(int(x)) {
			return 0, fmt.Errorf("expected int, got non-integral float64 %v", x)
		}
		return int(x), nil
	case string:
		n, err := strconv.Atoi(x)
		if err != nil {
			return 0, fmt.Errorf("expected int, got string %q: %v", x, err)
		}
		return n, nil
	}
	return 0, fmt.Errorf("expected int, got %T", v)
}

func asFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case uint64:
		return float64(x), nil
	case string:
		f, err := strconv.ParseFloat(x, 64)
		if err != nil {
			return 0, fmt.Errorf("expected float, got string %q: %v", x, err)
		}
		return f, nil
	}
	return 0, fmt.Errorf("expected float, got %T", v)
}

func asBool(v any) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		b, err := strconv.ParseBool(x)
		if err != nil {
			return false, fmt.Errorf("expected bool, got string %q: %v", x, err)
		}
		return b, nil
	}
	return false, fmt.Errorf("expected bool, got %T", v)
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

