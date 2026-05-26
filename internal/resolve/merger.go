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

	// Spell params are value-only overrides keyed by param name: the scroll can
	// override a param's default value, never its type. The type always comes
	// from source extraction.
	for name, value := range existing_spell.Params {
		for i := range function_descriptor.Params {
			if function_descriptor.Params[i].Name == name {
				function_descriptor.Params[i].Default = value
				break
			}
		}
	}

	if existing_spell.Interpreter != "" {
		function_descriptor.Interpreter = existing_spell.Interpreter
	}

	return nil
}