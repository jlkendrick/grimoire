package cmd

import (
	"os"
	"path/filepath"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	desc "github.com/jlkendrick/grimoire/internal/descriptor"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// TestLoadScrolls_GlobalRoutesToOwnCache pins down the multi-scroll global
// flow: when cwd is not upstream of any scroll, LoadScrolls returns one Scroll
// per registered_scrolls entry, each ReadDescriptorCache call resolves to a
// distinct on-disk cache file, and AddFunctionDescriptor writes to the file
// keyed off function_descriptor.ScrollPath without cross-contaminating
// sibling caches.
func TestLoadScrolls_GlobalRoutesToOwnCache(t *testing.T) {
	home := setupTestEnv(t)

	// Build two registered scroll directories outside of any cwd ancestor.
	scrollA := filepath.Join(home, "projectA", "scroll.yaml")
	scrollB := filepath.Join(home, "projectB", "scroll.yaml")
	if err := os.MkdirAll(filepath.Dir(scrollA), 0755); err != nil {
		t.Fatalf("mkdir A: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(scrollB), 0755); err != nil {
		t.Fatalf("mkdir B: %v", err)
	}
	writeFile(t, scrollA, "spells:\n  - command: alpha\n    path: alpha.py\n    function: alpha\n")
	writeFile(t, scrollB, "spells:\n  - command: beta\n    path: beta.py\n    function: beta\n")
	writeFile(t, filepath.Join(filepath.Dir(scrollA), "alpha.py"), "def alpha():\n    pass\n")
	writeFile(t, filepath.Join(filepath.Dir(scrollB), "beta.py"), "def beta():\n    pass\n")

	// Register both in the global grimoire.yaml.
	registry := &scroll.Scroll{
		RegisteredScrolls: []scroll.ScrollPath{{Path: scrollA}, {Path: scrollB}},
		Path:              filepath.Join(home, "grimoire.yaml"),
	}
	if err := registry.Write(); err != nil {
		t.Fatalf("write registry: %v", err)
	}

	// cwd in a directory not upstream of either scroll, forcing the global
	// fallback path inside LoadScrolls.
	isolated := filepath.Join(home, "elsewhere")
	if err := os.MkdirAll(isolated, 0755); err != nil {
		t.Fatalf("mkdir isolated: %v", err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(isolated); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	scrolls, err := scroll.LoadScrolls()
	if err != nil {
		t.Fatalf("LoadScrolls: %v", err)
	}
	if len(scrolls) != 2 {
		t.Fatalf("LoadScrolls returned %d scrolls, want 2", len(scrolls))
	}
	if scrolls[0].Path != scrollA || scrolls[1].Path != scrollB {
		t.Errorf("scroll order = [%s, %s], want [%s, %s]", scrolls[0].Path, scrolls[1].Path, scrollA, scrollB)
	}

	// Each scroll's cache should be backed by its own on-disk file. Force a
	// write through AddFunctionDescriptor for scroll A only and assert that
	// scroll B's cache file is unaffected.
	cacheA, err := cache.ReadDescriptorCache(scrollA)
	if err != nil {
		t.Fatalf("ReadDescriptorCache A: %v", err)
	}
	cacheB, err := cache.ReadDescriptorCache(scrollB)
	if err != nil {
		t.Fatalf("ReadDescriptorCache B: %v", err)
	}
	if cacheA == cacheB {
		t.Fatalf("ReadDescriptorCache returned the same DescriptorCache pointer for two scroll paths")
	}
	if cacheA.ScrollPath != scrollA || cacheB.ScrollPath != scrollB {
		t.Errorf("cache.ScrollPath mismatch: A=%s B=%s", cacheA.ScrollPath, cacheB.ScrollPath)
	}

	pathHashA, _ := utils.HashFilePath(scrollA)
	pathHashB, _ := utils.HashFilePath(scrollB)
	cacheFileA := filepath.Join(home, "cache", pathHashA+".json")
	cacheFileB := filepath.Join(home, "cache", pathHashB+".json")

	if err := cache.AddFunctionDescriptor(desc.FunctionDescriptor{
		CommandName:         "alpha",
		FunctionName:        "alpha",
		AbsPathToSourceFile: filepath.Join(filepath.Dir(scrollA), "alpha.py"),
		RelPathToSourceFile: "alpha.py",
		ScrollPath:          scrollA,
		SpellHash:           "fake-spell-hash-a",
		SourceHash:          "fake-source-hash-a",
	}); err != nil {
		t.Fatalf("AddFunctionDescriptor A: %v", err)
	}

	if _, err := os.Stat(cacheFileA); err != nil {
		t.Errorf("cache file for scroll A not written: %v", err)
	}
	if _, err := os.Stat(cacheFileB); !os.IsNotExist(err) {
		t.Errorf("cache file for scroll B should not exist after writing only A; stat err=%v", err)
	}

	// The in-memory cache for A should contain the new entry; B's should still be empty.
	cacheA2, err := cache.ReadDescriptorCache(scrollA)
	if err != nil {
		t.Fatalf("ReadDescriptorCache A (post-write): %v", err)
	}
	if _, ok := cacheA2.Functions["alpha"]; !ok {
		t.Errorf("post-write cache A missing 'alpha' descriptor")
	}
	cacheB2, err := cache.ReadDescriptorCache(scrollB)
	if err != nil {
		t.Fatalf("ReadDescriptorCache B (post-write): %v", err)
	}
	if len(cacheB2.Functions) != 0 {
		t.Errorf("cache B mutated by write to A: %+v", cacheB2.Functions)
	}
}
