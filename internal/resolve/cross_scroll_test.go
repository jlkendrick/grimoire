package resolve_test

import (
	"os"
	"strings"
	"testing"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// makeCache constructs an in-memory DescriptorCache containing the given
// spell command names. Source paths are bogus — these caches are used only
// to exercise the lookup logic, not the run-time engine.
func makeCache(scrollPath string, spells ...string) *cache.DescriptorCache {
	functions := map[string]descriptor.FunctionDescriptor{}
	for _, name := range spells {
		functions[name] = descriptor.FunctionDescriptor{
			CommandName:  name,
			FunctionName: name,
			ScrollPath:   scrollPath,
		}
	}
	return &cache.DescriptorCache{
		Version:    cache.CACHE_VERSION,
		ScrollPath: scrollPath,
		Functions:  functions,
		Pipelines:  map[string]descriptor.PipelineDescriptor{},
	}
}

func TestResolveSpellRef_BareLocal(t *testing.T) {
	local := makeCache("/a/scroll.yaml", "deploy")
	fd, err := resolve.ResolveSpellRef("deploy", local, nil)
	if err != nil {
		t.Fatalf("ResolveSpellRef: %v", err)
	}
	if fd.CommandName != "deploy" {
		t.Errorf("CommandName = %q, want %q", fd.CommandName, "deploy")
	}
}

func TestResolveSpellRef_BareLocalMissing(t *testing.T) {
	local := makeCache("/a/scroll.yaml", "deploy")
	_, err := resolve.ResolveSpellRef("nope", local, nil)
	if err == nil {
		t.Fatalf("expected error for missing local spell")
	}
	if !strings.Contains(err.Error(), "not found in local scroll") {
		t.Errorf("error = %v, want local-not-found", err)
	}
}

func TestResolveSpellRef_QualifiedCrossScroll(t *testing.T) {
	localA := makeCache("/a/scroll.yaml", "alpha")
	cacheB := makeCache("/b/scroll.yaml", "deploy")
	idx := resolve.BuildSpellIndex([]resolve.SpellIndexEntry{
		{Scroll: &scroll.Scroll{Name: "a", Path: "/a/scroll.yaml"}, Cache: localA},
		{Scroll: &scroll.Scroll{Name: "b", Path: "/b/scroll.yaml"}, Cache: cacheB},
	})

	fd, err := resolve.ResolveSpellRef("b.deploy", localA, idx)
	if err != nil {
		t.Fatalf("ResolveSpellRef: %v", err)
	}
	if fd.CommandName != "deploy" || fd.ScrollPath != "/b/scroll.yaml" {
		t.Errorf("resolved to %+v, want b.deploy from /b/scroll.yaml", fd)
	}
}

func TestResolveSpellRef_UnknownModule(t *testing.T) {
	localA := makeCache("/a/scroll.yaml", "alpha")
	idx := resolve.BuildSpellIndex([]resolve.SpellIndexEntry{
		{Scroll: &scroll.Scroll{Name: "a", Path: "/a/scroll.yaml"}, Cache: localA},
	})
	_, err := resolve.ResolveSpellRef("zzz.deploy", localA, idx)
	if err == nil {
		t.Fatalf("expected error for unknown module")
	}
	if !strings.Contains(err.Error(), "no scroll named") {
		t.Errorf("error = %v, want unknown-module", err)
	}
}

func TestResolveSpellRef_UnknownSpellInKnownModule(t *testing.T) {
	localA := makeCache("/a/scroll.yaml", "alpha")
	cacheB := makeCache("/b/scroll.yaml", "deploy")
	idx := resolve.BuildSpellIndex([]resolve.SpellIndexEntry{
		{Scroll: &scroll.Scroll{Name: "a", Path: "/a/scroll.yaml"}, Cache: localA},
		{Scroll: &scroll.Scroll{Name: "b", Path: "/b/scroll.yaml"}, Cache: cacheB},
	})

	_, err := resolve.ResolveSpellRef("b.missing", localA, idx)
	if err == nil {
		t.Fatalf("expected error for missing spell in known module")
	}
	if !strings.Contains(err.Error(), `has no spell "missing"`) {
		t.Errorf("error = %v, want missing-spell-in-module", err)
	}
}

func TestResolveSpellRef_RejectsMultiLevel(t *testing.T) {
	local := makeCache("/a/scroll.yaml", "x")
	_, err := resolve.ResolveSpellRef("a.b.c", local, nil)
	if err == nil {
		t.Fatalf("expected error for multi-level ref")
	}
	if !strings.Contains(err.Error(), "single-level qualification") {
		t.Errorf("error = %v, want single-level message", err)
	}
}

func TestResolveSpellRef_DottedWithoutIndex(t *testing.T) {
	local := makeCache("/a/scroll.yaml", "x")
	_, err := resolve.ResolveSpellRef("b.deploy", local, nil)
	if err == nil {
		t.Fatalf("expected error when SpellIndex is nil")
	}
	if !strings.Contains(err.Error(), "requires cross-scroll context") {
		t.Errorf("error = %v, want cross-scroll-needed", err)
	}
}

// captureStderr swaps os.Stderr for a pipe for the duration of fn and
// returns whatever was written. Used for the BuildSpellIndex shadow-warning
// assertions.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()

	fn()
	_ = w.Close()

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestBuildSpellIndex_DuplicateNameWarnsAndFirstWins(t *testing.T) {
	cacheA := makeCache("/a/scroll.yaml", "deploy")
	cacheB := makeCache("/b/scroll.yaml", "deploy_other")

	var idx *resolve.SpellIndex
	stderr := captureStderr(t, func() {
		idx = resolve.BuildSpellIndex([]resolve.SpellIndexEntry{
			{Scroll: &scroll.Scroll{Name: "shared", Path: "/a/scroll.yaml"}, Cache: cacheA},
			{Scroll: &scroll.Scroll{Name: "shared", Path: "/b/scroll.yaml"}, Cache: cacheB},
		})
	})

	if !strings.Contains(stderr, "shared by") {
		t.Errorf("expected stderr warning, got %q", stderr)
	}
	resolved := idx.ByScrollName["shared"]
	if resolved == nil {
		t.Fatalf("'shared' missing from index")
	}
	if resolved != cacheA {
		t.Errorf("'shared' resolved to %s, want first-registered (a)", resolved.ScrollPath)
	}
}

func TestBuildSpellIndex_SkipsEmptyName(t *testing.T) {
	registry := makeCache("/grimoire.yaml")
	idx := resolve.BuildSpellIndex([]resolve.SpellIndexEntry{
		{Scroll: &scroll.Scroll{Name: "", Path: "/grimoire.yaml"}, Cache: registry},
	})
	if len(idx.ByScrollName) != 0 {
		t.Errorf("empty-name scroll should be skipped; index = %+v", idx.ByScrollName)
	}
}
