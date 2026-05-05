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
	// Check if the scroll has changed since last run
	scroll_hash, err := utils.HashFile(scroll_obj.Path)
	if err != nil {
		return fmt.Errorf("error hashing scroll: %v", err)
	}
	if scroll_hash == descriptor_cache.ScrollHash {
		return nil
	}

	for _, spell := range scroll_obj.Spells {
		curr_hash, err := spell.Hash()
		if err != nil {
			return err
		}
		function_descriptor, ok := descriptor_cache.Functions[spell.Function]

		// If no cached descriptor or the scroll has changed, re-extract the function descriptor
		if !ok || curr_hash != function_descriptor.SpellHash {
			if !ok {
				fmt.Printf("%s Unearthed a new spell: %s. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(spell.Function))
			} else {
				fmt.Printf("%s Spell %s has changed since last run. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(spell.Function))
			}
			abs_path_to_function, err := utils.MakeScrollRelPathAbs(spell.Path, spell.ScrollPath)
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

			err = MergeSpellIntoFunctionDescriptor(spell, &resolved_descriptor)
			if err != nil {
				return fmt.Errorf("error merging spell into function descriptor: %v", err)
			}

			descriptor_cache.Functions[spell.Function] = resolved_descriptor
		}
	}

	// Prune cached descriptors for spells that are no longer in the scroll.
	in_scroll := make(map[string]struct{}, len(scroll_obj.Spells))
	for _, spell := range scroll_obj.Spells {
		in_scroll[spell.Function] = struct{}{}
	}
	for fn_name := range descriptor_cache.Functions {
		if _, ok := in_scroll[fn_name]; !ok {
			fmt.Printf("%s Banished spell %s from cache\n", utils.SpellStyle("-"), utils.SpellStyle(fn_name))
			delete(descriptor_cache.Functions, fn_name)
		}
	}

	// We got past the early return, so the scroll has diverged from the cached
	// hash. Persist once: refreshes ScrollHash and writes any extractions/prunes.
	if err := cache.WriteDescriptorCache(descriptor_cache); err != nil {
		return fmt.Errorf("error writing descriptor cache: %v", err)
	}
	return nil
}