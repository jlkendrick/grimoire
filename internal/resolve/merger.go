package resolve

import (
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

// Map spell fields to IR fields and merge them into the function descriptor
func MergeSpellIntoFunctionDescriptor(existing_spell scroll.Spell, function_descriptor *descriptor.FunctionDescriptor) error {
	function_descriptor.CommandName = existing_spell.Command

	if existing_spell.Path != "" {
		function_descriptor.RelPathToSourceFile = existing_spell.Path
	}

	if existing_spell.Function != "" {
		function_descriptor.FunctionName = existing_spell.Function
	}

	if len(existing_spell.Params) > 0 {
		for _, existing_param := range existing_spell.Params {
			// Look up param in function_descriptor.Params by name
			for i, ir_param := range function_descriptor.Params {
				if ir_param.Name == existing_param.Name {
					function_descriptor.Params[i].ResolvedType = &descriptor.TypeInfo{
						Name: existing_param.Type,
					}
					function_descriptor.Params[i].Default = existing_param.Default
					switch existing_param.Type {
					case "str", "int", "float", "bool":
						function_descriptor.Params[i].ResolvedType.Kind = descriptor.TypeKindPrimitive
					}
					break
				}
			}
		}
	}

	if existing_spell.Interpreter != "" {
		function_descriptor.Interpreter = existing_spell.Interpreter
	}

	return nil
}