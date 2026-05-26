package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// registerParamFlag adds a single typed flag to command from a ParamDescriptor.
// It dispatches on the param's resolved Kind:
//   - primitive (and optional-of-primitive) -> a typed scalar flag
//   - list of a primitive element            -> a repeatable cobra slice flag
//   - everything else (map, struct, list of  -> a string flag carrying a JSON
//     non-primitives, optional-of-complex,        literal, parsed in buildPayload
//     unknown)
//
// param.Default is the typed Go value of the parsed default literal. The
// asInt/asFloat/asBool/asString helpers absorb the type variance introduced
// by the JSON cache round-trip (numbers decoded into float64) and goccy
// go-yaml (which may produce uint64 for unsigned ints).
func registerParamFlag(command *cobra.Command, param descriptor.ParamDescriptor) error {
	kind, info := paramKind(param)

	switch kind {
	case descriptor.TypeKindPrimitive:
		return registerPrimitiveFlag(command, param, info.Name, param.Default == nil)

	case descriptor.TypeKindOptional:
		// Optionals are never required. An optional primitive still gets a typed
		// scalar flag; an optional complex type falls through to a JSON flag.
		if info.Element != nil && primitiveCategory(info.Element.Name) != "" {
			return registerPrimitiveFlag(command, param, info.Element.Name, false)
		}
		return registerJSONFlag(command, param, false)

	case descriptor.TypeKindList:
		if info.Element != nil && primitiveCategory(info.Element.Name) != "" {
			return registerSliceFlag(command, param, info.Element.Name)
		}
		return registerJSONFlag(command, param, param.Default == nil)

	default: // map, struct, unknown
		return registerJSONFlag(command, param, param.Default == nil)
	}
}

// registerPrimitiveFlag registers a scalar flag for one of the four primitive
// categories. typeName may be any of the language-specific spellings handled by
// primitiveCategory (e.g. "str"/"string", "int"/"int64", "float"/"float64").
func registerPrimitiveFlag(command *cobra.Command, param descriptor.ParamDescriptor, typeName string, required bool) error {
	switch primitiveCategory(typeName) {
	case "string":
		def := ""
		if param.Default != nil {
			s, err := asString(param.Default)
			if err != nil {
				return fmt.Errorf("default value for %s is not a string: %v", param.Name, err)
			}
			def = s
		}
		command.Flags().StringP(param.Name, "", def, "")

	case "int":
		def := 0
		if param.Default != nil {
			n, err := asInt(param.Default)
			if err != nil {
				return fmt.Errorf("default value for %s is not an int: %v", param.Name, err)
			}
			def = n
		}
		command.Flags().IntP(param.Name, "", def, "")

	case "bool":
		def := false
		if param.Default != nil {
			b, err := asBool(param.Default)
			if err != nil {
				return fmt.Errorf("default value for %s is not a bool: %v", param.Name, err)
			}
			def = b
		}
		command.Flags().BoolP(param.Name, "", def, "")

	case "float":
		def := 0.0
		if param.Default != nil {
			f, err := asFloat(param.Default)
			if err != nil {
				return fmt.Errorf("default value for %s is not a float: %v", param.Name, err)
			}
			def = f
		}
		command.Flags().Float64P(param.Name, "", def, "")

	default:
		return fmt.Errorf("unsupported primitive type: %s", typeName)
	}

	if required {
		command.MarkFlagRequired(param.Name)
	}
	return nil
}

// registerSliceFlag registers a repeatable cobra slice flag for a list of a
// primitive element type (e.g. --tags a --tags b, or --tags a,b).
func registerSliceFlag(command *cobra.Command, param descriptor.ParamDescriptor, elementType string) error {
	switch primitiveCategory(elementType) {
	case "string":
		command.Flags().StringSliceP(param.Name, "", toStringSlice(param.Default), "")
	case "int":
		command.Flags().IntSliceP(param.Name, "", toIntSlice(param.Default), "")
	case "bool":
		command.Flags().BoolSliceP(param.Name, "", toBoolSlice(param.Default), "")
	case "float":
		command.Flags().Float64SliceP(param.Name, "", toFloatSlice(param.Default), "")
	default:
		return fmt.Errorf("unsupported slice element type: %s", elementType)
	}
	if param.Default == nil {
		command.MarkFlagRequired(param.Name)
	}
	return nil
}

// registerJSONFlag registers a string flag whose value is parsed as a JSON
// literal in buildPayload. A complex default is serialized to its JSON form so
// it round-trips back through the parser.
func registerJSONFlag(command *cobra.Command, param descriptor.ParamDescriptor, required bool) error {
	def := ""
	if param.Default != nil {
		b, err := json.Marshal(param.Default)
		if err != nil {
			return fmt.Errorf("default value for %s is not serializable to JSON: %v", param.Name, err)
		}
		def = string(b)
	}
	command.Flags().StringP(param.Name, "", def, "")
	if required {
		command.MarkFlagRequired(param.Name)
	}
	return nil
}

// buildPayload reads the parsed flag values back into the args map that gets
// JSON-serialized for the runtime. It mirrors registerParamFlag's dispatch.
func buildPayload(function_descriptor descriptor.FunctionDescriptor, cmd *cobra.Command) (map[string]interface{}, error) {
	payload := make(map[string]interface{})

	for _, param := range function_descriptor.Params {
		kind, info := paramKind(param)

		switch kind {
		case descriptor.TypeKindPrimitive:
			readPrimitive(payload, cmd, param.Name, info.Name)

		case descriptor.TypeKindOptional:
			if info.Element != nil && primitiveCategory(info.Element.Name) != "" {
				readPrimitive(payload, cmd, param.Name, info.Element.Name)
			} else if err := readJSON(payload, cmd, param.Name); err != nil {
				return nil, err
			}

		case descriptor.TypeKindList:
			if info.Element != nil && primitiveCategory(info.Element.Name) != "" {
				readSlice(payload, cmd, param.Name, info.Element.Name)
			} else if err := readJSON(payload, cmd, param.Name); err != nil {
				return nil, err
			}

		default: // map, struct, unknown
			if err := readJSON(payload, cmd, param.Name); err != nil {
				return nil, err
			}
		}
	}

	return payload, nil
}

func readPrimitive(payload map[string]interface{}, cmd *cobra.Command, name, typeName string) {
	switch primitiveCategory(typeName) {
	case "int":
		val, _ := cmd.Flags().GetInt(name)
		payload[name] = val
	case "string":
		val, _ := cmd.Flags().GetString(name)
		payload[name] = val
	case "bool":
		val, _ := cmd.Flags().GetBool(name)
		payload[name] = val
	case "float":
		val, _ := cmd.Flags().GetFloat64(name)
		payload[name] = val
	}
}

func readSlice(payload map[string]interface{}, cmd *cobra.Command, name, elementType string) {
	switch primitiveCategory(elementType) {
	case "string":
		val, _ := cmd.Flags().GetStringSlice(name)
		payload[name] = val
	case "int":
		val, _ := cmd.Flags().GetIntSlice(name)
		payload[name] = val
	case "bool":
		val, _ := cmd.Flags().GetBoolSlice(name)
		payload[name] = val
	case "float":
		val, _ := cmd.Flags().GetFloat64Slice(name)
		payload[name] = val
	}
}

func readJSON(payload map[string]interface{}, cmd *cobra.Command, name string) error {
	raw, _ := cmd.Flags().GetString(name)
	if strings.TrimSpace(raw) == "" {
		// No value supplied and no default — leave it out of the payload.
		return nil
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return fmt.Errorf("param %s: invalid JSON value: %w", name, err)
	}
	payload[name] = decoded
	return nil
}

// paramKind resolves the effective TypeKind for a param. A nil ResolvedType or
// an empty Kind (a descriptor written before classification existed) falls back
// to inferring from the type Name, defaulting to Unknown.
func paramKind(param descriptor.ParamDescriptor) (descriptor.TypeKind, *descriptor.TypeInfo) {
	info := param.ResolvedType
	if info == nil {
		return descriptor.TypeKindUnknown, &descriptor.TypeInfo{Kind: descriptor.TypeKindUnknown}
	}
	if info.Kind != "" {
		return info.Kind, info
	}
	if primitiveCategory(info.Name) != "" {
		return descriptor.TypeKindPrimitive, info
	}
	return descriptor.TypeKindUnknown, info
}

// primitiveCategory normalizes the language-specific spellings of a scalar type
// into one of "string", "int", "bool", "float", or "" if it is not primitive.
func primitiveCategory(name string) string {
	switch name {
	case "string", "str":
		return "string"
	case "integer", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"rune", "byte":
		return "int"
	case "boolean", "bool":
		return "bool"
	case "float", "float32", "float64":
		return "float"
	}
	return ""
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

// toStringSlice / toIntSlice / toFloatSlice / toBoolSlice convert a list-typed
// default (typically []any from the JSON cache or goccy YAML) into the concrete
// slice cobra needs. A nil or unconvertible element yields a nil slice so the
// flag simply starts empty.
func toStringSlice(v any) []string {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		s, err := asString(it)
		if err != nil {
			return nil
		}
		out = append(out, s)
	}
	return out
}

func toIntSlice(v any) []int {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]int, 0, len(items))
	for _, it := range items {
		n, err := asInt(it)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

func toFloatSlice(v any) []float64 {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]float64, 0, len(items))
	for _, it := range items {
		f, err := asFloat(it)
		if err != nil {
			return nil
		}
		out = append(out, f)
	}
	return out
}

func toBoolSlice(v any) []bool {
	items, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]bool, 0, len(items))
	for _, it := range items {
		b, err := asBool(it)
		if err != nil {
			return nil
		}
		out = append(out, b)
	}
	return out
}
