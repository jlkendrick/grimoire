package cmd

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"

	desc "github.com/jlkendrick/grimoire/internal/descriptor"
)

// TestComplexFlags_RegisterAndBuildPayload covers the three non-primitive flag
// shapes end to end: a primitive list -> repeatable slice flag, a map and an
// element-less list -> JSON-literal string flags, and a complex default that
// round-trips through JSON serialization.
func TestComplexFlags_RegisterAndBuildPayload(t *testing.T) {
	fd := desc.FunctionDescriptor{
		CommandName: "party",
		Params: []desc.ParamDescriptor{
			{Name: "tags", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindList, Element: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "str"}, Name: "list[str]"}},
			{Name: "people", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindList, Name: "list"}},
			{Name: "meta", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindMap, Name: "dict"}, Default: map[string]any{"a": "b"}},
		},
	}

	cmd := &cobra.Command{Use: "party"}
	for _, p := range fd.Params {
		if err := registerParamFlag(cmd, p); err != nil {
			t.Fatalf("registerParamFlag(%s): %v", p.Name, err)
		}
	}

	if got := cmd.Flags().Lookup("tags").Value.Type(); got != "stringSlice" {
		t.Errorf("--tags type = %q, want stringSlice", got)
	}
	if got := cmd.Flags().Lookup("people").Value.Type(); got != "string" {
		t.Errorf("--people type = %q, want string", got)
	}
	metaFlag := cmd.Flags().Lookup("meta")
	if metaFlag.Value.Type() != "string" {
		t.Errorf("--meta type = %q, want string", metaFlag.Value.Type())
	}
	if metaFlag.DefValue != `{"a":"b"}` {
		t.Errorf("--meta default = %q, want JSON-serialized map", metaFlag.DefValue)
	}

	// Simulate user input.
	if err := cmd.Flags().Set("tags", "vip"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("tags", "new"); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Flags().Set("people", `[{"Name":"james"}]`); err != nil {
		t.Fatal(err)
	}

	payload, err := buildPayload(fd, cmd)
	if err != nil {
		t.Fatalf("buildPayload: %v", err)
	}

	if !reflect.DeepEqual(payload["tags"], []string{"vip", "new"}) {
		t.Errorf("tags = %#v, want [vip new]", payload["tags"])
	}
	wantPeople := []interface{}{map[string]interface{}{"Name": "james"}}
	if !reflect.DeepEqual(payload["people"], wantPeople) {
		t.Errorf("people = %#v, want %#v", payload["people"], wantPeople)
	}
	wantMeta := map[string]interface{}{"a": "b"}
	if !reflect.DeepEqual(payload["meta"], wantMeta) {
		t.Errorf("meta = %#v, want %#v (from default)", payload["meta"], wantMeta)
	}
}

// TestJSONFlag_InvalidValueErrors confirms a malformed JSON literal surfaces a
// clear error rather than being silently dropped.
func TestJSONFlag_InvalidValueErrors(t *testing.T) {
	fd := desc.FunctionDescriptor{
		Params: []desc.ParamDescriptor{
			{Name: "meta", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindMap, Name: "dict"}},
		},
	}
	cmd := &cobra.Command{Use: "x"}
	if err := registerParamFlag(cmd, fd.Params[0]); err != nil {
		t.Fatalf("registerParamFlag: %v", err)
	}
	if err := cmd.Flags().Set("meta", "not json"); err != nil {
		t.Fatal(err)
	}
	if _, err := buildPayload(fd, cmd); err == nil {
		t.Errorf("expected error for invalid JSON, got nil")
	}
}
