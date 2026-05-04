package resolve

import (
	"fmt"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	cache "github.com/jlkendrick/grimoire/internal/cache"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	extract "github.com/jlkendrick/grimoire/internal/extract"
)

// Check if any of the cached descriptors are stale relative to the spell entries in the user's scroll.yaml file
func ReconcileScrollAndFunctionDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache) error {
	for _, spell := range scroll_obj.Spells {
		curr_hash, err := spell.Hash()
		if err != nil {
			return err
		}
		function_descriptor, ok := descriptor_cache.Functions[spell.Function]
		if !ok {
			return fmt.Errorf("function descriptor not found in cache: %s", spell.Function)
		}
		if curr_hash != function_descriptor.SpellHash {
			fmt.Printf("%s Spell has changed since last run. Updating runtime config...\n", utils.AccentStyle("+"))
			// Re-extract the function descriptor
			abs_path_to_function, err := utils.MakeScrollRelPathAbs(spell.Path, scroll_obj.Path)
			if err != nil {
				return fmt.Errorf("error making scroll rel path abs: %v", err)
			}
			function_descriptor_generator := extract.FunctionDescriptorGenerator{
				CommandName: spell.Command,
				FunctionName: spell.Function,
				RelPathToSourceFile: spell.Path,
				AbsPathToSourceFile: abs_path_to_function,
				ScrollPath: scroll_obj.Path,
				SpellHash: curr_hash,
				Interpreter: spell.Interpreter,
			}
			resolved_descriptor, err := function_descriptor_generator.Generate()
			if err != nil {
				return fmt.Errorf("error generating spell descriptor: %v", err)
			}

			// Resolve the descriptor
			err = ResolveFunctionDescriptor(&resolved_descriptor)
			if err != nil {
				return fmt.Errorf("error resolving function descriptor: %v", err)
			}

			// Write the updated descriptor to the cache
			err = cache.AddFunctionDescriptor(resolved_descriptor)
			if err != nil {
				return fmt.Errorf("error writing descriptor cache: %v", err)
			}
			descriptor_cache.Functions[spell.Function] = resolved_descriptor
		}
	}
	return nil
}