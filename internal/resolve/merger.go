package resolve

import (
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	ir "github.com/jlkendrick/grimoire/internal/ir"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// MergeSpellIntoFunctionDescriptor is a delegate kept for the old call
// sites (cmd/add) until they move to the ir conversion entry points.
//
// Deprecated: use ir.MergeSpell.
func MergeSpellIntoFunctionDescriptor(existing_spell scroll.Spell, function_descriptor *descriptor.FunctionDescriptor) error {
	return ir.MergeSpell(existing_spell, function_descriptor)
}
