package resolve

import (
	"fmt"
	"slices"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	expr "github.com/jlkendrick/grimoire/internal/expr"
	extract "github.com/jlkendrick/grimoire/internal/extract"
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

// getStepReference returns the root binding id of a step-param reference
// like "step.field" or "step[0]" — empty string if the value is not a
// reference. Crucially, bare strings without an accessor are NOT
// references (they're literal user-provided values); reference detection
// only kicks in when a path accessor is present.
func getStepReference(value any) string {
	s, ok := value.(string)
	if !ok {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '.' || s[i] == '[' {
			return s[:i]
		}
	}
	return ""
}

// ValidateRitual runs the same validation the pipeline reconciler performs —
// first-step-is-spell check, lexical scope checks on step-id references, and
// condition-expression parsing for let/if steps — against the supplied
// descriptor cache. It does not touch the cache or scroll: callers (e.g. the
// weave transpiler) use it to confirm a freshly-built ritual would survive
// the next reconcile before writing it to the scroll.
//
// Pass a nil SpellIndex when the ritual is known to use bare references
// only; dotted refs will error in that case.
func ValidateRitual(ritual scroll.Ritual, descriptor_cache *cache.DescriptorCache, index *SpellIndex) error {
	if len(ritual.Steps) > 0 && ritual.Steps[0].Kind() != "spell" {
		return fmt.Errorf("ritual %s: first step must be a spell (got %s; CLI flags are derived from the entry spell's params)", ritual.Command, ritual.Steps[0].Kind())
	}
	_, err := validateAndConvertSteps(ritual.Steps, nil, descriptor_cache, index, ritual.Command)
	return err
}

func ReconcileScrollAndPipelineDescriptors(scroll_obj *scroll.Scroll, descriptor_cache *cache.DescriptorCache, index *SpellIndex) (bool, error) {
	mutated := false

	// Rituals -> Pipeline descriptors
	for _, ritual := range scroll_obj.Rituals {
		if len(ritual.Steps) > 0 && ritual.Steps[0].Kind() != "spell" {
			return false, fmt.Errorf("ritual %s: first step must be a spell (got %s; CLI flags are derived from the entry spell's params)", ritual.Command, ritual.Steps[0].Kind())
		}

		// Recursively validate steps and convert to descriptors. The walker
		// enforces lexical scope for step-id references: a step can see
		// ancestor-scope ids and earlier-sibling ids, but not ids declared
		// inside a sibling branch.
		steps, err := validateAndConvertSteps(ritual.Steps, nil, descriptor_cache, index, ritual.Command)
		if err != nil {
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
			descriptor_cache.Pipelines[ritual.Command] = descriptor.PipelineDescriptor{
				CommandName: ritual.Command,
				Steps:       steps,
				RitualHash:  new_hash,
			}
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

// validateAndConvertSteps walks a step list, enforces per-step rules, and
// returns the corresponding descriptor steps. inheritedIds carries the
// step ids visible from ancestor scopes (NOT sibling-branch scopes —
// branch-internal ids stay branch-internal). index supplies cross-scroll
// spell lookups for dotted refs; nil rejects all dotted refs.
func validateAndConvertSteps(steps []scroll.Step, inheritedIds []string, descriptor_cache *cache.DescriptorCache, index *SpellIndex, ritualName string) ([]descriptor.StepDescriptor, error) {
	out := make([]descriptor.StepDescriptor, 0, len(steps))
	siblingIds := []string{}

	visible := func() []string {
		v := make([]string, 0, len(inheritedIds)+len(siblingIds))
		v = append(v, inheritedIds...)
		v = append(v, siblingIds...)
		return v
	}

	for _, step := range steps {
		switch step.Kind() {
		case "spell":
			if _, err := ResolveSpellRef(step.Spell, descriptor_cache, index); err != nil {
				return nil, fmt.Errorf("ritual %s: %v", ritualName, err)
			}
			for _, value := range step.Params {
				if ref := getStepReference(value); ref != "" && !slices.Contains(visible(), ref) {
					return nil, fmt.Errorf("ritual %s: step %s references %s which is not in scope", ritualName, step.Spell, ref)
				}
			}
			out = append(out, descriptor.StepDescriptor{
				Id:        step.Id,
				SpellName: step.Spell,
				Params:    step.Params,
			})
			if step.Id != "" {
				siblingIds = append(siblingIds, step.Id)
			}

		case "let":
			if step.Spell != "" || step.If != "" || len(step.Then) > 0 || len(step.Else) > 0 {
				return nil, fmt.Errorf("ritual %s: let-step %q cannot mix with spell/if fields", ritualName, step.Let)
			}
			if step.Id != "" {
				return nil, fmt.Errorf("ritual %s: let-step uses 'let:' for the binding name; remove the redundant 'id:' field", ritualName)
			}
			if step.Value == "" {
				return nil, fmt.Errorf("ritual %s: let %q requires a 'value:' expression", ritualName, step.Let)
			}
			letExpr, err := expr.ParseCondition(step.Value)
			if err != nil {
				return nil, fmt.Errorf("ritual %s: invalid let %q value %q: %v", ritualName, step.Let, step.Value, err)
			}
			for _, root := range expr.RootRefs(letExpr) {
				if !slices.Contains(visible(), root) {
					return nil, fmt.Errorf("ritual %s: let %q references %s which is not in scope", ritualName, step.Let, root)
				}
			}
			out = append(out, descriptor.StepDescriptor{
				Let:   step.Let,
				Value: step.Value,
			})
			siblingIds = append(siblingIds, step.Let)

		case "if":
			if step.Id != "" {
				return nil, fmt.Errorf("ritual %s: if-step cannot have an id", ritualName)
			}
			if step.Spell != "" {
				return nil, fmt.Errorf("ritual %s: step cannot set both 'if' and 'spell'", ritualName)
			}
			if len(step.Then) == 0 {
				return nil, fmt.Errorf("ritual %s: if-step 'then' branch cannot be empty", ritualName)
			}
			condExpr, err := expr.ParseCondition(step.If)
			if err != nil {
				return nil, fmt.Errorf("ritual %s: invalid condition %q: %v", ritualName, step.If, err)
			}
			for _, root := range expr.RootRefs(condExpr) {
				if !slices.Contains(visible(), root) {
					return nil, fmt.Errorf("ritual %s: condition references %s which is not in scope", ritualName, root)
				}
			}

			branchScope := visible()
			thenSteps, err := validateAndConvertSteps(step.Then, branchScope, descriptor_cache, index, ritualName)
			if err != nil {
				return nil, err
			}
			var elseSteps []descriptor.StepDescriptor
			if len(step.Else) > 0 {
				elseSteps, err = validateAndConvertSteps(step.Else, branchScope, descriptor_cache, index, ritualName)
				if err != nil {
					return nil, err
				}
			}

			out = append(out, descriptor.StepDescriptor{
				Condition: step.If,
				Then:      thenSteps,
				Else:      elseSteps,
			})
		}
	}

	return out, nil
}
