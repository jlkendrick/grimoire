package resolve_test

import (
	"reflect"
	"strings"
	"testing"

	resolve "github.com/jlkendrick/grimoire/internal/resolve"
)

func TestResolveReference_NonStringPassesThrough(t *testing.T) {
	bindings := map[string]any{"step1": "anything"}
	got, ok, err := resolve.ResolveReference(42, bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if ok {
		t.Errorf("ok = true, want false for non-string input")
	}
	if got != 42 {
		t.Errorf("got %v, want 42", got)
	}
}

func TestResolveReference_StringWithNoMatchingBindingPassesThrough(t *testing.T) {
	bindings := map[string]any{"step1": "anything"}
	got, ok, err := resolve.ResolveReference("just a plain string", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if ok {
		t.Errorf("ok = true, want false for non-reference string")
	}
	if got != "just a plain string" {
		t.Errorf("got %v, want plain string back", got)
	}
}

func TestResolveReference_BareBindingNameIsAReference(t *testing.T) {
	// A bare binding name (no accessor) is treated as a reference
	// to the full result of the previous step.
	bindings := map[string]any{"step1": "hello"}
	got, ok, err := resolve.ResolveReference("step1", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true for bare binding name")
	}
	if got != "hello" {
		t.Errorf("got %v, want hello", got)
	}
}

func TestResolveReference_ListIndex(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{"a", "b", "c"},
	}
	got, ok, err := resolve.ResolveReference("step1[1]", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "b" {
		t.Errorf("got %v, want b", got)
	}
}

func TestResolveReference_MapKey(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{"name": "alice", "age": 30},
	}
	got, ok, err := resolve.ResolveReference("step1.name", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "alice" {
		t.Errorf("got %v, want alice", got)
	}
}

func TestResolveReference_ListThenKey(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{
			map[string]any{"name": "alice"},
			map[string]any{"name": "bob"},
		},
	}
	got, ok, err := resolve.ResolveReference("step1[1].name", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "bob" {
		t.Errorf("got %v, want bob", got)
	}
}

func TestResolveReference_KeyThenIndex(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{
			"items": []any{"x", "y", "z"},
		},
	}
	got, ok, err := resolve.ResolveReference("step1.items[2]", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "z" {
		t.Errorf("got %v, want z", got)
	}
}

func TestResolveReference_NestedMapKeys(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{
			"user": map[string]any{
				"profile": map[string]any{
					"name": "carol",
				},
			},
		},
	}
	got, ok, err := resolve.ResolveReference("step1.user.profile.name", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "carol" {
		t.Errorf("got %v, want carol", got)
	}
}

func TestResolveReference_NestedListIndexes(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{
			[]any{"a", "b"},
			[]any{"c", "d"},
		},
	}
	got, ok, err := resolve.ResolveReference("step1[1][0]", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "c" {
		t.Errorf("got %v, want c", got)
	}
}

func TestResolveReference_DeepMixedChain(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{
			"users": []any{
				map[string]any{
					"name": "alice",
					"tags": []any{"admin", "ops"},
				},
				map[string]any{
					"name": "bob",
					"tags": []any{"dev"},
				},
			},
		},
	}
	got, ok, err := resolve.ResolveReference("step1.users[0].tags[1]", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	if got != "ops" {
		t.Errorf("got %v, want ops", got)
	}
}

func TestResolveReference_ResolvesToNonScalar(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{
			"items": []any{"x", "y"},
		},
	}
	got, ok, err := resolve.ResolveReference("step1.items", bindings)
	if err != nil {
		t.Fatalf("ResolveReference: %v", err)
	}
	if !ok {
		t.Errorf("ok = false, want true")
	}
	want := []any{"x", "y"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolveReference_IndexOutOfBounds(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{"a", "b"},
	}
	_, _, err := resolve.ResolveReference("step1[5]", bindings)
	if err == nil {
		t.Fatalf("expected error for out-of-bounds index")
	}
	if !strings.Contains(err.Error(), "out of bounds") {
		t.Errorf("error = %q, want it to mention out of bounds", err)
	}
}

func TestResolveReference_NegativeIndexRejected(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{"a", "b"},
	}
	_, _, err := resolve.ResolveReference("step1[-1]", bindings)
	if err == nil {
		t.Fatalf("expected error for negative index")
	}
}

func TestResolveReference_NonNumericIndex(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{"a", "b"},
	}
	_, _, err := resolve.ResolveReference("step1[abc]", bindings)
	if err == nil {
		t.Fatalf("expected error for non-numeric index")
	}
	if !strings.Contains(err.Error(), "invalid index") {
		t.Errorf("error = %q, want it to mention invalid index", err)
	}
}

func TestResolveReference_IndexOnNonList(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{"name": "alice"},
	}
	_, _, err := resolve.ResolveReference("step1[0]", bindings)
	if err == nil {
		t.Fatalf("expected error when indexing a non-list")
	}
	if !strings.Contains(err.Error(), "not a list") {
		t.Errorf("error = %q, want it to mention not a list", err)
	}
}

func TestResolveReference_KeyAccessOnNonMap(t *testing.T) {
	bindings := map[string]any{
		"step1": []any{"a", "b"},
	}
	_, _, err := resolve.ResolveReference("step1.name", bindings)
	if err == nil {
		t.Fatalf("expected error when key-accessing a non-map")
	}
	if !strings.Contains(err.Error(), "not a map") {
		t.Errorf("error = %q, want it to mention not a map", err)
	}
}

func TestResolveReference_MissingKey(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{"name": "alice"},
	}
	_, _, err := resolve.ResolveReference("step1.missing", bindings)
	if err == nil {
		t.Fatalf("expected error for missing key")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want it to mention not found", err)
	}
}

func TestResolveReference_MidChainKeyMissing(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{
			"user": map[string]any{"name": "alice"},
		},
	}
	_, _, err := resolve.ResolveReference("step1.user.missing", bindings)
	if err == nil {
		t.Fatalf("expected error for missing key in chain")
	}
}

func TestResolveReference_MidChainIndexOutOfBounds(t *testing.T) {
	bindings := map[string]any{
		"step1": map[string]any{
			"items": []any{"a"},
		},
	}
	_, _, err := resolve.ResolveReference("step1.items[7]", bindings)
	if err == nil {
		t.Fatalf("expected error for out-of-bounds in chain")
	}
}
