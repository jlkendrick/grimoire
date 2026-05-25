package cmd

import (
	"fmt"
	"os"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// loadedScroll bundles a parsed scroll with its reconciled descriptor cache
// so the multi-pass resolver doesn't have to re-read the cache between passes.
type loadedScroll struct {
	scroll *scroll.Scroll
	cache  *cache.DescriptorCache
}

// reconcileSpellsAcrossScrolls reads each scroll's descriptor cache and
// reconciles its spells. Rituals are NOT reconciled here — they happen in a
// second pass once all spells are fresh. Returns the loaded entries in
// input order, suitable for building a SpellIndex.
func reconcileSpellsAcrossScrolls(scrolls []*scroll.Scroll) ([]loadedScroll, error) {
	loaded := make([]loadedScroll, 0, len(scrolls))
	for _, s := range scrolls {
		c, err := cache.ReadDescriptorCache(s.Path)
		if err != nil {
			return nil, fmt.Errorf("loading cache for %s: %v", s.Path, err)
		}
		if err := resolve.ReconcileSpellsOnly(s, c); err != nil {
			return nil, err
		}
		loaded = append(loaded, loadedScroll{scroll: s, cache: c})
	}
	return loaded, nil
}

// buildSpellIndexFor turns reconciled scroll/cache pairs into a SpellIndex
// keyed by scroll Name. Same-name collisions emit a stderr warning and the
// first scroll wins.
func buildSpellIndexFor(loaded []loadedScroll) *resolve.SpellIndex {
	entries := make([]resolve.SpellIndexEntry, 0, len(loaded))
	for _, ld := range loaded {
		entries = append(entries, resolve.SpellIndexEntry{Scroll: ld.scroll, Cache: ld.cache})
	}
	return resolve.BuildSpellIndex(entries)
}

// buildGlobalSpellIndex loads the union of local and registered scrolls,
// reconciles their spells, and returns a SpellIndex usable for cross-scroll
// ritual ref validation. Exposed as a helper for ad-hoc callers (e.g. the
// weave write flow) that need cross-scroll context but don't go through
// registerScrollCommands.
func buildGlobalSpellIndex() (*resolve.SpellIndex, []loadedScroll, error) {
	_, all, err := scroll.LoadActiveAndAll()
	if err != nil {
		return nil, nil, err
	}
	loaded, err := reconcileSpellsAcrossScrolls(all)
	if err != nil {
		return nil, nil, err
	}
	return buildSpellIndexFor(loaded), loaded, nil
}

// registerScrollCommands runs the full multi-pass flow:
//
//  1. Load active scrolls (for CLI registration) and all scrolls (for the
//     cross-scroll spell index).
//  2. Reconcile spells across every scroll in the union.
//  3. Build the SpellIndex.
//  4. Reconcile rituals against the index (cross-scroll refs validated here).
//  5. Compute command-name disambiguation and register cobra commands for
//     active scrolls only.
//
// `active` defaults to the slice returned from `scroll.LoadActiveAndAll`.
// We accept it as an argument so cmd/root.go can keep its current
// load-and-call shape and tests can drive registerScrollCommands directly.
func registerScrollCommands(active []*scroll.Scroll) error {
	// Build the union: active scrolls plus everything in the registry not
	// already in active. We can't call scroll.LoadActiveAndAll() here
	// because callers (and tests) might pass an arbitrary `active` slice
	// that bypasses the registry; for those, treat active itself as the
	// union.
	all := active
	if union, err := loadAllForActive(active); err == nil {
		all = union
	}

	loaded, err := reconcileSpellsAcrossScrolls(all)
	if err != nil {
		return err
	}
	index := buildSpellIndexFor(loaded)

	// Reconcile rituals across every scroll in the union. Cross-scroll
	// refs are validated against `index`.
	for _, ld := range loaded {
		if err := resolve.ReconcileRitualsOnly(ld.scroll, ld.cache, index); err != nil {
			return err
		}
	}

	// Now register cobra commands for ACTIVE scrolls only. Non-active
	// scrolls were loaded only to populate the SpellIndex.
	activeSet := map[string]struct{}{}
	for _, s := range active {
		activeSet[s.Path] = struct{}{}
	}

	commandSources := map[string][]*scroll.Scroll{}
	activeLoaded := make([]loadedScroll, 0, len(active))
	for _, ld := range loaded {
		if _, ok := activeSet[ld.scroll.Path]; !ok {
			continue
		}
		if ld.cache.Functions == nil && ld.cache.Pipelines == nil {
			continue
		}
		activeLoaded = append(activeLoaded, ld)
		for cmdName := range ld.cache.Functions {
			commandSources[cmdName] = append(commandSources[cmdName], ld.scroll)
		}
		for cmdName := range ld.cache.Pipelines {
			commandSources[cmdName] = append(commandSources[cmdName], ld.scroll)
		}
	}

	seen := map[string]string{}
	for _, ld := range activeLoaded {
		resolvedNames := map[string]string{}
		for cmdName := range ld.cache.Functions {
			resolvedNames[cmdName] = resolveCommandName(ld.scroll, cmdName, commandSources)
		}
		for cmdName := range ld.cache.Pipelines {
			resolvedNames[cmdName] = resolveCommandName(ld.scroll, cmdName, commandSources)
		}

		cmds, err := GenerateCommands(ld.cache, resolvedNames, ld.scroll.Path, index)
		if err != nil {
			return fmt.Errorf("generating commands for %s: %v", ld.scroll.Path, err)
		}
		for _, c := range cmds {
			if prev, ok := seen[c.Use]; ok {
				fmt.Fprintf(os.Stderr, "warning: command %q in %s shadowed by earlier definition in %s\n", c.Use, ld.scroll.Path, prev)
				continue
			}
			seen[c.Use] = ld.scroll.Path
			rootCmd.AddCommand(c)
		}
	}
	return nil
}

// loadAllForActive returns the union of the given active scrolls and every
// scroll in the global registry. Returns an error if the registry can't be
// loaded — the caller can decide to fall back to active-only.
func loadAllForActive(active []*scroll.Scroll) ([]*scroll.Scroll, error) {
	global, err := scroll.LoadGlobalScrolls()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	union := make([]*scroll.Scroll, 0, len(active)+len(global))
	for _, s := range active {
		if _, ok := seen[s.Path]; ok {
			continue
		}
		seen[s.Path] = struct{}{}
		union = append(union, s)
	}
	for _, s := range global {
		if _, ok := seen[s.Path]; ok {
			continue
		}
		seen[s.Path] = struct{}{}
		union = append(union, s)
	}
	return union, nil
}

// resolveCommandName decides whether a command should be registered bare or
// as a `<scroll-name>.<command>` dot-form. Dot-form applies when (a) two or
// more scrolls define the command, or (b) the command name matches a
// reserved static command (whose bare form would otherwise be unreachable
// because the static-command short-circuit at startup wins).
func resolveCommandName(s *scroll.Scroll, command string, sources map[string][]*scroll.Scroll) string {
	if staticCommands[command] || len(sources[command]) > 1 {
		return s.Name + "." + command
	}
	return command
}
