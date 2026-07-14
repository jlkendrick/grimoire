package resolve

import (
	"fmt"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	extract "github.com/jlkendrick/grimoire/internal/extract"
	graph "github.com/jlkendrick/grimoire/internal/graph"
	ir "github.com/jlkendrick/grimoire/internal/ir"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// Reconcile the function descriptor with the source code
func ReconcileFunctionDescriptor(function_descriptor *descriptor.FunctionDescriptor) (descriptor.FunctionDescriptor, error) {
	source_hash, err := utils.HashFile(function_descriptor.AbsPathToSourceFile)
	if err != nil {
		return *function_descriptor, fmt.Errorf("error hashing source file: %v", err)
	}
	if source_hash != function_descriptor.SourceHash {
		fmt.Printf("%s Source code change detected. Updating casting recipe...\n", utils.SpellStyle("+"))
		function_descriptor_generator := extract.FunctionDescriptorGenerator{
			CommandName:         function_descriptor.CommandName,
			FunctionName:        function_descriptor.FunctionName,
			RelPathToSourceFile: function_descriptor.RelPathToSourceFile,
			AbsPathToSourceFile: function_descriptor.AbsPathToSourceFile,
			ScrollPath:          function_descriptor.ScrollPath,
			SpellHash:           function_descriptor.SpellHash, // Is fresh since root command checks for staleness
			Interpreter:         function_descriptor.Interpreter,
		}
		resolved_descriptor, err := function_descriptor_generator.Generate()
		if err != nil {
			return *function_descriptor, fmt.Errorf("error generating spell descriptor: %v", err)
		}

		if err := cache.AddFunctionDescriptor(resolved_descriptor); err != nil {
			return *function_descriptor, fmt.Errorf("error caching function descriptor: %v", err)
		}
		return resolved_descriptor, nil
	}

	return *function_descriptor, nil
}

// ReconcileScrollAndDescriptors performs the full single-scroll reconcile
// (spells + rituals) against the supplied cache. Cross-scroll ritual refs
// are not supported here — callers that need them should use the two-phase
// flow via ReconcileSpellsOnly / ReconcileRitualsOnly with a SpellIndex.
func ReconcileScrollAndDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache) error {
	return reconcileScroll(scroll_obj, descriptor_cache, nil)
}

// ReconcileSpellsOnly reconciles just the spell entries of a scroll into
// its descriptor cache. Used by the two-phase reconcile in dispatch so all
// spells across all scrolls are fresh before any ritual is validated.
func ReconcileSpellsOnly(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache) error {
	mutated, err := ReconcileScrollAndFunctionDescriptors(scroll_obj, descriptor_cache)
	if err != nil {
		return fmt.Errorf("error reconciling scroll and function descriptors: %v", err)
	}
	if mutated {
		if err := cache.WriteDescriptorCache(descriptor_cache); err != nil {
			return fmt.Errorf("error writing descriptor cache: %v", err)
		}
	}
	return nil
}

// ReconcileRitualsOnly reconciles just the ritual entries of a scroll
// against the supplied spell index. Bare step refs resolve against the
// scroll's local cache; dotted refs (e.g. `b.deploy`) resolve against
// index.ByScrollName[b].
func ReconcileRitualsOnly(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache, index *SpellIndex) error {
	mutated, err := ReconcileScrollAndPipelineDescriptors(scroll_obj, descriptor_cache, index)
	if err != nil {
		return fmt.Errorf("error reconciling scroll and pipeline descriptors: %v", err)
	}
	if mutated {
		if err := cache.WriteDescriptorCache(descriptor_cache); err != nil {
			return fmt.Errorf("error writing descriptor cache: %v", err)
		}
	}
	return nil
}

func reconcileScroll(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache, index *SpellIndex) error {
	scroll_hash, err := utils.HashFile(scroll_obj.Path)
	if err != nil {
		return fmt.Errorf("error hashing scroll: %v", err)
	}
	if scroll_hash == descriptor_cache.ScrollHash {
		return nil
	}

	mutated1, err := ReconcileScrollAndFunctionDescriptors(scroll_obj, descriptor_cache)
	if err != nil {
		return fmt.Errorf("error reconciling scroll and function descriptors: %v", err)
	}

	mutated2, err := ReconcileScrollAndPipelineDescriptors(scroll_obj, descriptor_cache, index)
	if err != nil {
		return fmt.Errorf("error reconciling scroll and pipeline descriptors: %v", err)
	}

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
			// Conversion is ir.FromSpell's job (extraction + override
			// merge); this loop only decides WHEN it runs and caches the
			// result.
			resolved_descriptor, err := ir.FromSpell(spell)
			if err != nil {
				return false, fmt.Errorf("error generating spell descriptor: %v", err)
			}

			descriptor_cache.Functions[spell.Command] = *resolved_descriptor
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

// ValidateRitual reports whether a ritual would survive reconcile: lower
// it to IR and validate by building — the same front half the reconciler
// runs at scroll load. Callers (e.g. the weave transpiler) use it to
// confirm a freshly-built ritual before writing it to the scroll. Pass a
// nil SpellIndex when the ritual uses bare references only.
func ValidateRitual(ritual scroll.Ritual, descriptor_cache *cache.DescriptorCache, index *SpellIndex) error {
	return graph.ValidatePipeline(ir.FromRitual(&ritual), spellExists(descriptor_cache, index))
}

// spellExists adapts spell-ref resolution to the existence check
// graph.ValidatePipeline expects.
func spellExists(descriptor_cache *cache.DescriptorCache, index *SpellIndex) func(name string) error {
	return func(name string) error {
		_, err := ResolveSpellRef(name, descriptor_cache, index)
		return err
	}
}

func ReconcileScrollAndPipelineDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache, index *SpellIndex) (bool, error) {
	mutated := false

	// Rituals -> Pipelines. Conversion (ir.FromRitual) answers what the
	// ritual means; validation (graph.ValidatePipeline) answers whether it
	// would build — the same builder the runtime uses, so scroll-load
	// errors match run-time reality. This loop only decides WHEN and
	// caches the result.
	for _, ritual := range scroll_obj.Rituals {
		pipeline := ir.FromRitual(&ritual)
		if err := graph.ValidatePipeline(pipeline, spellExists(descriptor_cache, index)); err != nil {
			return false, err
		}

		new_hash, err := ritual.Hash()
		if err != nil {
			return false, fmt.Errorf("error hashing ritual: %v", err)
		}
		old_hash := ""
		old_pipeline, ok := descriptor_cache.Pipelines[ritual.Command]
		if ok {
			old_hash = old_pipeline.RitualHash
		}
		if !ok || new_hash != old_hash {
			if !ok {
				fmt.Printf("%s Unearthed a new ritual: %s. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(ritual.Command))
			} else {
				fmt.Printf("%s Ritual %s has changed since last run. Divining signature...\n", utils.SpellStyle("+"), utils.SpellStyle(ritual.Command))
			}
			pipeline.RitualHash = new_hash
			descriptor_cache.Pipelines[ritual.Command] = *pipeline
			mutated = true
		}
	}

	// Prune cached descriptors for rituals that are no longer in the scroll.
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
