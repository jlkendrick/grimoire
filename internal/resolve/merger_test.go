package resolve_test

import (
	"testing"

	desc "github.com/jlkendrick/grimoire/internal/descriptor"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

func TestMerge_OverlaysCommandAndFunction(t *testing.T) {
	d := desc.FunctionDescriptor{
		CommandName:         "old_cmd",
		FunctionName:        "old_fn",
		RelPathToSourceFile: "old.py",
		Interpreter:         "old_py",
	}
	s := scroll.Spell{
		Command:     "new_cmd",
		Function:    "new_fn",
		Path:        "new.py",
		Interpreter: "new_py",
	}
	if err := resolve.MergeSpellIntoFunctionDescriptor(s, &d); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if d.CommandName != "new_cmd" {
		t.Errorf("CommandName = %q, want new_cmd", d.CommandName)
	}
	if d.FunctionName != "new_fn" {
		t.Errorf("FunctionName = %q, want new_fn", d.FunctionName)
	}
	if d.RelPathToSourceFile != "new.py" {
		t.Errorf("RelPathToSourceFile = %q, want new.py", d.RelPathToSourceFile)
	}
	if d.Interpreter != "new_py" {
		t.Errorf("Interpreter = %q, want new_py", d.Interpreter)
	}
}

// TestMerge_EmptySpellPathLeavesDescriptorPath guards merger.go L12: an empty
// spell.Path must NOT clobber the descriptor's existing path. The same logic
// applies to Function and Interpreter; Path is the canonical case.
func TestMerge_EmptySpellPathLeavesDescriptorPath(t *testing.T) {
	d := desc.FunctionDescriptor{
		CommandName:         "x",
		RelPathToSourceFile: "extracted.py",
		FunctionName:        "x",
		Interpreter:         "py3",
	}
	s := scroll.Spell{Command: "x"} // Path, Function, Interpreter all empty

	if err := resolve.MergeSpellIntoFunctionDescriptor(s, &d); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if d.RelPathToSourceFile != "extracted.py" {
		t.Errorf("RelPathToSourceFile = %q, want extracted.py (empty spell.Path must not overwrite)", d.RelPathToSourceFile)
	}
	if d.FunctionName != "x" {
		t.Errorf("FunctionName = %q, want x", d.FunctionName)
	}
	if d.Interpreter != "py3" {
		t.Errorf("Interpreter = %q, want py3", d.Interpreter)
	}
}

func TestMerge_ParamTypeAndDefaultOverlay(t *testing.T) {
	d := desc.FunctionDescriptor{
		Params: []desc.ParamDescriptor{
			{Name: "n", ResolvedType: &desc.TypeInfo{Kind: desc.TypeKindPrimitive, Name: "int"}, Default: nil},
		},
	}
	s := scroll.Spell{
		Command: "c",
		Params:  []scroll.Param{{Name: "n", Type: "int", Default: "5"}},
	}
	if err := resolve.MergeSpellIntoFunctionDescriptor(s, &d); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if d.Params[0].ResolvedType == nil || d.Params[0].ResolvedType.Name != "int" {
		t.Errorf("ResolvedType.Name = %+v, want int", d.Params[0].ResolvedType)
	}
	if d.Params[0].Default != "5" {
		t.Errorf("Default = %#v, want %q", d.Params[0].Default, "5")
	}
}

// TestMerge_UnmatchedSpellParamIgnored: a spell-declared param whose name
// doesn't match any extracted param should be silently ignored — not
// inserted, not panic.
func TestMerge_UnmatchedSpellParamIgnored(t *testing.T) {
	d := desc.FunctionDescriptor{
		Params: []desc.ParamDescriptor{
			{Name: "name", ResolvedType: &desc.TypeInfo{Name: "str"}, Default: "world"},
		},
	}
	s := scroll.Spell{
		Command: "c",
		Params:  []scroll.Param{{Name: "ghost", Type: "int", Default: "1"}},
	}
	if err := resolve.MergeSpellIntoFunctionDescriptor(s, &d); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(d.Params) != 1 {
		t.Errorf("Params length changed: %d, want 1", len(d.Params))
	}
	if d.Params[0].Name != "name" || d.Params[0].Default != "world" {
		t.Errorf("matched param mutated unexpectedly: %+v", d.Params[0])
	}
}

func TestMerge_NoParamsLeavesDescriptorParams(t *testing.T) {
	d := desc.FunctionDescriptor{
		Params: []desc.ParamDescriptor{
			{Name: "name", ResolvedType: &desc.TypeInfo{Name: "str"}, Default: "world"},
		},
	}
	original := d.Params[0]
	s := scroll.Spell{Command: "c"} // no Params

	if err := resolve.MergeSpellIntoFunctionDescriptor(s, &d); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if len(d.Params) != 1 {
		t.Fatalf("Params length changed: %d, want 1", len(d.Params))
	}
	if d.Params[0].Default != original.Default {
		t.Errorf("Default mutated: %v -> %v", original.Default, d.Params[0].Default)
	}
}
