package cache_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	desc "github.com/jlkendrick/grimoire/internal/descriptor"
	ts "github.com/jlkendrick/grimoire/internal/testsupport"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// scrollPathOnDisk creates an empty scroll.yaml at <dir>/scroll.yaml so that
// utils.HashFile (called by WriteDescriptorCache) succeeds, and returns its path.
func scrollPathOnDisk(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "scroll.yaml")
	ts.WriteFile(t, p, "spells: []\n")
	return p
}

func cacheFileFor(t *testing.T, scrollPath string) string {
	t.Helper()
	home, err := utils.GrimoireHome()
	if err != nil {
		t.Fatalf("GrimoireHome: %v", err)
	}
	pathHash, err := utils.HashFilePath(scrollPath)
	if err != nil {
		t.Fatalf("HashFilePath: %v", err)
	}
	return filepath.Join(home, "cache", pathHash+".json")
}

func TestReadDescriptorCache_NewReturnsEmpty(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	scrollPath := scrollPathOnDisk(t, dir)

	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	if dc == nil {
		t.Fatal("nil cache")
	}
	if dc.Version != cache.CACHE_VERSION {
		t.Errorf("Version = %d, want %d", dc.Version, cache.CACHE_VERSION)
	}
	if dc.ScrollPath != scrollPath {
		t.Errorf("ScrollPath = %q, want %q", dc.ScrollPath, scrollPath)
	}
	if dc.ScrollHash != "" {
		t.Errorf("ScrollHash should be empty so reconciler runs; got %q", dc.ScrollHash)
	}
	if dc.Functions == nil {
		t.Errorf("Functions map nil; should be initialized for safe writes")
	}
	if len(dc.Functions) != 0 {
		t.Errorf("expected empty Functions map, got %d entries", len(dc.Functions))
	}
}

func TestWriteThenReadRoundTrip(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	scrollPath := scrollPathOnDisk(t, dir)

	dc, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	dc.Functions["greet"] = desc.FunctionDescriptor{
		CommandName:         "greet",
		FunctionName:        "greet",
		AbsPathToSourceFile: filepath.Join(dir, "greet.py"),
		RelPathToSourceFile: "greet.py",
		ScrollPath:          scrollPath,
		Interpreter:         "python3",
		SourceHash:          "src-hash",
		SpellHash:           "spell-hash",
		Params: []desc.ParamDescriptor{
			{Name: "name", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "str"}, Default: "world"},
		},
	}

	if err := cache.WriteDescriptorCache(dc); err != nil {
		t.Fatalf("WriteDescriptorCache: %v", err)
	}

	cache.ResetCache()
	got, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache after reset: %v", err)
	}

	wantHash, err := utils.HashFile(scrollPath)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	if got.ScrollHash != wantHash {
		t.Errorf("ScrollHash after Write = %q, want %q", got.ScrollHash, wantHash)
	}
	if !reflect.DeepEqual(got.Functions["greet"], dc.Functions["greet"]) {
		t.Errorf("round-trip mismatch:\n got: %+v\nwant: %+v", got.Functions["greet"], dc.Functions["greet"])
	}
}

func TestAddFunctionDescriptor_PersistsToDisk(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	scrollPath := scrollPathOnDisk(t, dir)

	fd := desc.FunctionDescriptor{
		CommandName:  "greet",
		FunctionName: "greet",
		ScrollPath:   scrollPath,
		Params:       []desc.ParamDescriptor{{Name: "name"}},
	}
	if err := cache.AddFunctionDescriptor(fd); err != nil {
		t.Fatalf("AddFunctionDescriptor: %v", err)
	}

	cache.ResetCache()
	got, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	stored, ok := got.Functions["greet"]
	if !ok {
		t.Fatalf("descriptor not persisted under CommandName 'greet'; map: %+v", got.Functions)
	}
	if stored.FunctionName != "greet" {
		t.Errorf("FunctionName = %q, want %q", stored.FunctionName, "greet")
	}
}

// TestReadDescriptorCache_NilMapsInJSONInitialized guards the safety net at
// encoding.go L88-95: if the JSON on disk has "functions": null (e.g. a cache
// file that never had any descriptors written), the read path must still
// initialize non-nil maps so subsequent inserts don't panic.
func TestReadDescriptorCache_NilMapsInJSONInitialized(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	scrollPath := scrollPathOnDisk(t, dir)

	cacheFile := cacheFileFor(t, scrollPath)
	body, err := json.Marshal(map[string]any{
		"version":     cache.CACHE_VERSION,
		"scroll_hash": "",
		"scroll_path": scrollPath,
		"functions":   nil,
		"pipelines":   nil,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(cacheFile, body, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cache.ResetCache()
	got, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache: %v", err)
	}
	if got.Functions == nil {
		t.Errorf("Functions map nil after reading null JSON; would panic on insert")
	}
	if got.Pipelines == nil {
		t.Errorf("Pipelines map nil after reading null JSON; would panic on insert")
	}
}

func TestResetCache_ClearsMemo(t *testing.T) {
	ts.SetupGrimoireHome(t)
	dir := ts.WithScrollDir(t)
	scrollPath := scrollPathOnDisk(t, dir)

	first, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache 1: %v", err)
	}
	again, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache 2: %v", err)
	}
	if first != again {
		t.Errorf("expected memoized identity, got distinct pointers")
	}

	cache.ResetCache()
	fresh, err := cache.ReadDescriptorCache(scrollPath)
	if err != nil {
		t.Fatalf("ReadDescriptorCache 3: %v", err)
	}
	if first == fresh {
		t.Errorf("expected fresh pointer after reset, got the same one")
	}
}
