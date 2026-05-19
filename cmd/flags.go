package cmd

import (
	"fmt"
	"bytes"
	"strconv"
	"encoding/json"

	"github.com/spf13/cobra"

	resolve "github.com/jlkendrick/grimoire/internal/resolve"
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

// buildPayloadFromResult turns the previous step's stdout (a single JSON
// value) into a payload for the next step. If the previous output is a JSON
// list whose length matches the next step's param count and there is more
// than one param, it unpacks positionally — mirroring Python's
// `return val1, val2`. If the previous output is a JSON map, it assigns values 
// based on matching keys and param names. If all params are mapped, use that payload.
// Otherwise the whole decoded value is bound to the first param. Single-param steps
// never destructure, so a function that returns a list-as-data reaches the next step intact.
func buildPayloadFromResult(prev_output []byte, function_descriptor descriptor.FunctionDescriptor) (map[string]interface{}, error) {
	var decoded interface{}
	if len(bytes.TrimSpace(prev_output)) > 0 {
		if err := json.Unmarshal(prev_output, &decoded); err != nil {
			return nil, fmt.Errorf("step %s: previous output is not valid JSON: %v", function_descriptor.CommandName, err)
		}
	}

	fmt.Printf("decoded: %v\n", decoded)

	payload := make(map[string]interface{})
	if list, ok := decoded.([]interface{}); ok && len(function_descriptor.Params) > 1 {
		if len(list) != len(function_descriptor.Params) {
			return nil, fmt.Errorf("step %s: previous output has %d values but step expects %d params", function_descriptor.CommandName, len(list), len(function_descriptor.Params))
		}
		for i, param := range function_descriptor.Params {
			payload[param.Name] = list[i]
		}
		return payload, nil
	} else if _map, ok := decoded.(map[string]interface{}); ok {
		fmt.Printf("map: %v\n", _map)
		mapped_params := 0
		for _, param := range function_descriptor.Params {
			if value, ok := _map[param.Name]; ok {
				payload[param.Name] = value
				mapped_params++
			}
		}
		if mapped_params == len(function_descriptor.Params) {
			return payload, nil
		}
	}

	payload = make(map[string]interface{})

	if len(function_descriptor.Params) >= 1 {
		payload[function_descriptor.Params[0].Name] = decoded
	}
	return payload, nil
}

func buildPayloadFromBindings(params map[string]any, bindings map[string]any, function_descriptor descriptor.FunctionDescriptor) (map[string]interface{}, error) {
	payload := make(map[string]interface{})
	
	for name, value := range params {
		// Check if the value is a reference to a previous step output
		ref_value, is_reference, err := resolve.ResolveReference(value, bindings)
		if err != nil {
			return nil, err
		}
		if is_reference {
			payload[name] = ref_value
		} else {
			// It's a literal value
			payload[name] = value
		}
	}

	// Fill in any missing parameters with the function descriptor's default values
	for _, param := range function_descriptor.Params {
		if _, ok := payload[param.Name]; !ok {
			if param.Default != nil {
				payload[param.Name] = param.Default
			}
		}
	}

	return payload, nil
}