package resolve

import (
	"fmt"
	"os"
	"strings"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// SpellIndex maps a scroll's Name to its reconciled descriptor cache so
// ritual steps can resolve cross-scroll references of the form
// `<scroll-name>.<spell>`. Scrolls with no Name (e.g. the global registry
// file itself) are not represented.
type SpellIndex struct {
	ByScrollName map[string]*cache.DescriptorCache
}

// SpellIndexEntry pairs a scroll with its already-reconciled cache.
type SpellIndexEntry struct {
	Scroll *scroll.Scroll
	Cache  *cache.DescriptorCache
}

// BuildSpellIndex walks the entries in order and registers each scroll's
// cache under its Name. When two scrolls share a Name, the first wins and a
// warning is emitted to stderr — matching the CLI command-shadow behavior
// for same-name scrolls. Scrolls without a Name are skipped.
func BuildSpellIndex(entries []SpellIndexEntry) *SpellIndex {
	idx := &SpellIndex{ByScrollName: map[string]*cache.DescriptorCache{}}
	seenPath := map[string]string{}
	for _, e := range entries {
		if e.Scroll == nil || e.Scroll.Name == "" {
			continue
		}
		if prev, ok := seenPath[e.Scroll.Name]; ok {
			fmt.Fprintf(os.Stderr, "warning: scroll name %q is shared by %s and %s; cross-scroll refs to %q will resolve to %s\n", e.Scroll.Name, prev, e.Scroll.Path, e.Scroll.Name, prev)
			continue
		}
		idx.ByScrollName[e.Scroll.Name] = e.Cache
		seenPath[e.Scroll.Name] = e.Scroll.Path
	}
	return idx
}

// ResolveSpellRef looks up a ritual step's spell reference against the
// step's local cache (for bare names) or the SpellIndex (for dotted
// `<scroll-name>.<spell>` refs). Returns the resolved descriptor or a
// detailed error explaining what was missing.
//
// Only one level of qualification is supported — `a.b.c` errors out. This
// mirrors the CLI dot-notation convention introduced for command
// disambiguation.
func ResolveSpellRef(stepSpell string, localCache *cache.DescriptorCache, index *SpellIndex) (*descriptor.FunctionDescriptor, error) {
	dot := strings.IndexByte(stepSpell, '.')
	if dot < 0 {
		if localCache == nil {
			return nil, fmt.Errorf("spell %q: no local cache available", stepSpell)
		}
		fd, ok := localCache.Functions[stepSpell]
		if !ok {
			return nil, fmt.Errorf("spell %q not found in local scroll", stepSpell)
		}
		return &fd, nil
	}

	module := stepSpell[:dot]
	rest := stepSpell[dot+1:]
	if strings.IndexByte(rest, '.') >= 0 {
		return nil, fmt.Errorf("spell reference %q: only single-level qualification supported (use `module.spell`)", stepSpell)
	}
	if module == "" || rest == "" {
		return nil, fmt.Errorf("spell reference %q: empty module or spell name", stepSpell)
	}
	if index == nil {
		return nil, fmt.Errorf("spell reference %q requires cross-scroll context, but none was provided", stepSpell)
	}
	moduleCache, ok := index.ByScrollName[module]
	if !ok {
		return nil, fmt.Errorf("spell reference %q: no scroll named %q is registered", stepSpell, module)
	}
	fd, ok := moduleCache.Functions[rest]
	if !ok {
		return nil, fmt.Errorf("spell reference %q: scroll %q has no spell %q", stepSpell, module, rest)
	}
	return &fd, nil
}
