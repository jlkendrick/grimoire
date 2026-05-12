package resolve

import (
	"fmt"

	utils "github.com/jlkendrick/grimoire/internal/utils"
	cache "github.com/jlkendrick/grimoire/internal/cache"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	extract "github.com/jlkendrick/grimoire/internal/extract"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func ReconcileScrollAndDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache) error {
	// Check if the scroll has changed since last run (early return if not)
	scroll_hash, err := utils.HashFile(scroll_obj.Path)
	if err != nil {
		return fmt.Errorf("error hashing scroll: %v", err)
	}
	if scroll_hash == descriptor_cache.ScrollHash {
		return nil
	}

	// Reconcile the scroll and function descriptors
	mutated1, err := ReconcileScrollAndFunctionDescriptors(scroll_obj, descriptor_cache); if err != nil {
		return fmt.Errorf("error reconciling scroll and function descriptors: %v", err)
	}

	// Reconcile the scroll and pipeline descriptors
	mutated2, err := ReconcileScrollAndPipelineDescriptors(scroll_obj, descriptor_cache); if err != nil {
		return fmt.Errorf("error reconciling scroll and pipeline descriptors: %v", err)
	}

	// If we made any changes, write the descriptor cache
	if mutated1 || mutated2 {
		descriptor_cache.ScrollHash = scroll_hash
		if err := cache.WriteDescriptorCache(descriptor_cache); err != nil {
			return fmt.Errorf("error writing descriptor cache: %v", err)
		}
	}
	return nil
}

// Check if any of the cached descriptors are stale relative to the spell entries in the user's scroll.yaml file
func ReconcileScrollAndFunctionDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache) (bool, error) {
	mutated := false

	// Spells -> Function descriptors
	for _, spell := range scroll_obj.Spells {
		curr_hash, err := spell.Hash()
		if err != nil {
			return false, fmt.Errorf("error hashing spell: %v", err)
		}
		function_descriptor, ok := descriptor_cache.Functions[spell.Command]

		// If no cached descriptor or the scroll has changed, re-extract the function descriptor
		if !ok || curr_hash != function_descriptor.SpellHash {
			if !ok {
				fmt.Printf("%s Unearthed a new spell: %s. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(spell.Command))
			} else {
				fmt.Printf("%s Spell %s has changed since last run. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(spell.Command))
			}
			abs_path_to_function, err := utils.MakeScrollRelPathAbs(spell.Path, spell.ScrollPath)
			if err != nil {
				return false, fmt.Errorf("error making scroll rel path abs: %v", err)
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
				return false, fmt.Errorf("error generating spell descriptor: %v", err)
			}

			err = MergeSpellIntoFunctionDescriptor(spell, &resolved_descriptor)
			if err != nil {
				return false, fmt.Errorf("error merging spell into function descriptor: %v", err)
			}

			descriptor_cache.Functions[spell.Command] = resolved_descriptor
			mutated = true
		}
	}

	// Prune cached descriptors for spells that are no longer in the scroll.
	in_scroll := make(map[string]struct{}, len(scroll_obj.Spells))
	for _, spell := range scroll_obj.Spells {
		in_scroll[spell.Command] = struct{}{}
	}
	for fn_name := range descriptor_cache.Functions {
		if _, ok := in_scroll[fn_name]; !ok {
			fmt.Printf("%s Banished spell %s from cache\n", utils.SpellStyle("-"), utils.SpellStyle(fn_name))
			delete(descriptor_cache.Functions, fn_name)
			mutated = true
		}
	}
	return mutated, nil
}

func ReconcileScrollAndPipelineDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache) (bool, error) {
	mutated := false

	// Rituals -> Pipeline descriptors
	for _, ritual := range scroll_obj.Rituals {
		fmt.Printf("Reconciling ritual %s\n", ritual.Command)
		// All we have to do here is check that all steps in the ritual are valid spells
		// in our descriptor cache. They are already reconciled by the above function that
		// runs before this one in the root.go file.
		for _, step := range ritual.Steps {
			_, ok := descriptor_cache.Functions[step.Spell]
			if !ok {
				return false, fmt.Errorf("spell %s not found in descriptor cache", step.Spell)
			}
		}

		// If the ritual is not in the descriptor cache or has been updated since last run, add it
		new_hash, err := ritual.Hash()
		if err != nil {
			return false, fmt.Errorf("error hashing ritual: %v", err)
		}
		old_hash := ""
		old_pipeline, ok := descriptor_cache.Pipelines[ritual.Command]
		if ok {
			old_hash = old_pipeline.RitualHash
		}
		if _, ok := descriptor_cache.Pipelines[ritual.Command]; !ok || new_hash != old_hash {
			if !ok {
				fmt.Printf("%s Unearthed a new ritual: %s. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(ritual.Command))
			} else {
				fmt.Printf("%s Ritual %s has changed since last run. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(ritual.Command))
			}
			steps := make([]descriptor.StepDescriptor, 0, len(ritual.Steps))
			for _, step := range ritual.Steps {
				steps = append(steps, descriptor.StepDescriptor{
					Id: step.Id,
					SpellName: step.Spell,
					Params: step.Params,
				})
			}
			ritual_hash, err := ritual.Hash()
			if err != nil {
				return false, fmt.Errorf("error hashing ritual: %v", err)
			}
			pipeline_descriptor := descriptor.PipelineDescriptor{
				CommandName: ritual.Command,
				Steps: steps,
				RitualHash: ritual_hash,
			}
			descriptor_cache.Pipelines[ritual.Command] = pipeline_descriptor
			mutated = true
		}
	}

	// Prune cached descriptors for functions that are no longer in the scroll.
	in_scroll := make(map[string]struct{}, len(scroll_obj.Rituals))
	for _, ritual := range scroll_obj.Rituals {
		in_scroll[ritual.Command] = struct{}{}
	}
	for ritual_name := range descriptor_cache.Pipelines {
		if _, ok := in_scroll[ritual_name]; !ok {
			fmt.Printf("%s Banished ritual %s from cache\n", utils.SpellStyle("-"), utils.SpellStyle(ritual_name))
			delete(descriptor_cache.Pipelines, ritual_name)
			mutated = true
		}
	}

	return mutated, nil
}