package expr

import (
	"fmt"
	"strconv"
	"strings"
)

func ResolveReference(value any, bindings map[string]any) (any, bool, error) {
	isReference := func(value any) bool {
		// If the value is not a string, it's not a reference
		if _, ok := value.(string); !ok {
			return false
		}
		value_str := value.(string)
		for id := range bindings {
			if id == value_str || // Exact match
				strings.HasPrefix(value_str, id+"[") || // List indexing
				strings.HasPrefix(value_str, id+".") || // Object key access
				strings.Contains(value_str, id+"$") || // Did list indexing
				strings.Contains(value_str, id+"@") { // Did object key access
				return true
			}
		}
		return false
	}

	getFirstAccessorIdx := func(value_str string, start_idx int) int {
		for i := start_idx; i < len(value_str); i++ {
			c := value_str[i]
			if c == '.' || c == '[' {
				return i
			}
		}
		return len(value_str)
	}

	if !isReference(value) {
		return value, false, nil
	}

	// Try to extract the id or from the value
	value_str, ok := value.(string)
	if !ok {
		return value, false, nil
	}

	// Base case for recursion: if there is a binding for the value, return it
	if binding, ok := bindings[value_str]; ok {
		return binding, true, nil
	}

	// Get the first '.' or '[' in the value string
	first_accessor_idx := getFirstAccessorIdx(value_str, 0)
	accessor := value_str[first_accessor_idx]
	switch accessor {
	case '[':
		// List indexing
		id := strings.Split(value_str, "[")[0]
		ref_output, ok := bindings[id]
		if !ok {
			return nil, false, fmt.Errorf("reference %s not found in bindings", id)
		}
		// Get the index from the value
		left_bracket_idx := first_accessor_idx
		right_bracket_idx := strings.Index(value_str, "]")
		index := value_str[left_bracket_idx+1 : right_bracket_idx]
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

		// Recurse to resolve multiple levels of accessors

		// step_id[N].field
		// bindings[step_id$N$] = bindings[step_id][N]
		// then, run ResolveReference(step_id$N$.field, bindings) to resolve the next level

		// Add a binding for what we just resolved
		binding_id := id + "$" + index + "$"
		bindings[binding_id] = ref_value
		return ResolveReference(binding_id+value_str[right_bracket_idx+1:], bindings)

	case '.':
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
		key_left := first_accessor_idx + 1 // Skip the '.'
		key_right := getFirstAccessorIdx(value_str, key_left)
		key := value_str[key_left:key_right]
		ref_value, ok := ref_output_map[key]
		if !ok {
			return nil, false, fmt.Errorf("key %s not found in reference %s", key, id)
		}

		// Recurse to resolve multiple levels of accessors

		// step_id.field[N]
		// bindings[step_id@field] = bindings[step_id]["field"]
		// then, run ResolveReference(step_id@field, bindings) to resolve the next level

		// Add a binding for what we just resolved
		binding_id := id + "@" + key
		bindings[binding_id] = ref_value
		return ResolveReference(binding_id+value_str[key_right:], bindings)
	}

	// What the
	return value, false, nil
}
