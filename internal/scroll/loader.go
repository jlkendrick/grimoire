package scroll

import (
	"fmt"
	"os"
	"path/filepath"

	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// cached_scrolls memoizes parsed Scroll objects within a single CLI invocation,
// keyed by absolute path. Cobra command execution is single-threaded, so a
// plain map is safe.
var cached_scrolls = map[string]*Scroll{}

// ResetScrollCache clears all cached scroll state. Intended for use in tests.
func ResetScrollCache() {
	cached_scrolls = map[string]*Scroll{}
}

// LoadScrolls returns the active set of scrolls for this invocation:
//   - If a scroll.yaml is found by walking upward from cwd, returns a single-
//     element slice containing that local scroll.
//   - Otherwise, parses ~/.grimoire/grimoire.yaml and returns one Scroll per
//     RegisteredScrolls entry, in registry order (see LoadGlobalScrolls)
func LoadScrolls() ([]*Scroll, error) {
	if local, found, err := LoadLocalScroll(); err != nil {
		return nil, err
	} else if found {
		return []*Scroll{local}, nil
	}

	return LoadGlobalScrolls()
}

// LoadGlobalScrolls loads the global scroll registry from $GRIMOIRE_HOME/grimoire.yaml.
func LoadGlobalScrolls() ([]*Scroll, error) {
	registry, err := LoadRegistry()
	if err != nil {
		return nil, err
	}
	
	scrolls := make([]*Scroll, 0, len(registry.RegisteredScrolls))
	for _, sp := range registry.RegisteredScrolls {
		// If the scroll no longer exists, skip it
		if _, err := os.Stat(sp.Path); os.IsNotExist(err) {
			continue
		}
		s, err := loadScrollFile(sp.Path)
		if err != nil {
			return nil, fmt.Errorf("loading registered scroll %s: %w", sp.Path, err)
		}
		scrolls = append(scrolls, s)
	}

	return scrolls, nil
}


// LoadLocalScroll walks upward from cwd looking for a scroll.yaml. Returns
// (scroll, true, nil) on hit, (nil, false, nil) on miss.
func LoadLocalScroll() (*Scroll, bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, false, err
	}
	path, found := FindLocalScroll(cwd)
	if !found {
		return nil, false, nil
	}
	s, err := loadScrollFile(path)
	if err != nil {
		return nil, false, err
	}
	return s, true, nil
}

// LoadRegistry parses the global ~/.grimoire/grimoire.yaml file. The returned
// Scroll's Path points at grimoire.yaml so callers can mutate
// RegisteredScrolls and Write() the result back.
func LoadRegistry() (*Scroll, error) {
	home, err := utils.GrimoireHome()
	if err != nil {
		return nil, err
	}
	return loadScrollFile(filepath.Join(home, "grimoire.yaml"))
}

func loadScrollFile(path string) (*Scroll, error) {
	if cached, ok := cached_scrolls[path]; ok {
		return cached, nil
	}
	s, err := ParseScroll(path)
	if err != nil {
		return nil, err
	}
	cached_scrolls[path] = s
	return s, nil
}


// resolveAddScroll returns the local scroll that `add` should write into. If
// no scroll exists in cwd or any parent, initializes one in cwd and registers
// it with the global grimoire.
func ResolveAddScroll() (*Scroll, error) {
	if local, found, err := LoadLocalScroll(); err != nil {
		return nil, fmt.Errorf("Error loading local scroll: %v", err)
	} else if found {
		return local, nil
	}

	current_dir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("Error getting current directory: %v", err)
	}

	fmt.Printf("%s No scroll found, initializing new scroll\n", utils.AccentStyle("+"))
	cfg, err := InitScroll(current_dir, false)
	if err != nil {
		return nil, fmt.Errorf("Error initializing scroll: %v", err)
	}
	fmt.Printf("%s Inscribed scroll.yaml\n  · %s\n", utils.AccentStyle("+"), utils.DimStyle(cfg.Path))
	if err := RegisterScroll(cfg.Path); err != nil {
		return nil, fmt.Errorf("Error registering scroll: %v", err)
	}
	fmt.Printf("%s Bound %s to the global grimoire\n", utils.AccentStyle("+"), cfg.Path)
	return cfg, nil
}