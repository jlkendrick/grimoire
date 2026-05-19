package resolve

import (
	"fmt"
	"strings"
	"strconv"

	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func ResolveFunctionDescriptor(fd *descriptor.FunctionDescriptor) error {
	// TODO: implement methods for resolving types.
	// Simple casting is already done in the cmd/commands.go file.
	return nil
}

func ResolveReference(value any, bindings map[string]any) (any, bool, error) {
	isReference := func(value any) bool {
		// If the value is not a string, it's not a reference
		if _, ok := value.(string); !ok {
			return false
		}
		value_str := value.(string)
		for id := range bindings {
			if strings.HasPrefix(value_str, id+"[") || // List indexing
				strings.HasPrefix(value_str, id+".") { // Object key access
				return true
			}
		}
		return false	
	}
	if !isReference(value) {
		return value, false, nil
	}

	// Try to extract the id or from the value
	value_str, ok := value.(string)
	if !ok {
		return value, false, nil
	}
	if strings.Contains(value_str, "[") {
		// List indexing
		id := strings.Split(value_str, "[")[0]
		ref_output, ok := bindings[id]
		if !ok {
			return nil, false, fmt.Errorf("reference %s not found in bindings", id)
		}
		// Get the index from the value
		left_bracket_idx := strings.Index(value_str, "[")
		right_bracket_idx := strings.Index(value_str, "]")
		index := value_str[left_bracket_idx+1:right_bracket_idx]
		index_int, err := strconv.Atoi(index)
		if err != nil {
			return nil, false, fmt.Errorf("invalid index %s", index)
		}
		ref_output_list, ok := ref_output.([]any)
		if !ok {
			return nil, false, fmt.Errorf("reference %s is not a list", id)
		}
		// Check if the index is out of bounds
		if index_int < 0 || index_int >= len(ref_output_list) {
			return nil, false, fmt.Errorf("index %s is out of bounds", index)
		}
		ref_value := ref_output_list[index_int]
		
		// No recursion; ref_output is already resolved
		return ref_value, true, nil

	} else if strings.Contains(value_str, ".") {
		// Object key access
		id := strings.Split(value_str, ".")[0]
		ref_output, ok := bindings[id]
		if !ok {
			return nil, false, fmt.Errorf("reference %s not found in bindings", id)
		}
		ref_output_map, ok := ref_output.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("reference %s is not a map", id)
		}
		key := strings.Split(value_str, ".")[1]
		ref_value, ok := ref_output_map[key]
		if !ok {
			return nil, false, fmt.Errorf("key %s not found in reference %s", key, id)
		}
		return ref_value, true, nil
	}

	// What the
	return value, false, nil
}