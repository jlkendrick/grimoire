package ir

import (
	"fmt"

	extract "github.com/jlkendrick/grimoire/internal/extract"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// FromRitual lowers a user-model ritual into an IR pipeline. It is a pure
// structural mapping — no validation, no I/O, cannot fail: a malformed
// ritual lowers fine and is rejected by validation
// (graph.ValidatePipeline). Keeping conversion dumb means "what does this
// scroll mean" is decided in exactly one trivially testable place, and
// validation/sync never hide inside it.
func FromRitual(r *scroll.Ritual) *Pipeline {
	return &Pipeline{
		Command: r.Command,
		Steps:   fromSteps(r.Steps),
	}
}

// FromSpell resolves a user-model spell into an IR function. Unlike
// FromRitual this cannot be a pure mapping: a ritual's meaning lives
// entirely in the scroll, but half a spell's meaning — its signature —
// lives in the source file. Lowering a spell therefore runs signature
// extraction (file I/O + parsing) and can fail. Precedence: types always
// come from source; the scroll's value-only param entries override
// extracted defaults; an explicit interpreter wins.
//
// The sync layer decides WHEN to call this (hash comparisons) and caches
// the result; FromSpell itself only answers WHAT the spell means.
func FromSpell(s scroll.Spell) (*Function, error) {
	spellHash, err := s.Hash()
	if err != nil {
		return nil, fmt.Errorf("hash spell %s: %w", s.Command, err)
	}
	absPath, err := utils.MakeScrollRelPathAbs(s.Path, s.ScrollPath)
	if err != nil {
		return nil, fmt.Errorf("resolve source path for spell %s: %w", s.Command, err)
	}

	gen := extract.FunctionDescriptorGenerator{
		CommandName:         s.Command,
		FunctionName:        s.Function,
		RelPathToSourceFile: s.Path,
		AbsPathToSourceFile: absPath,
		ScrollPath:          s.ScrollPath,
		SpellHash:           spellHash,
		Interpreter:         s.Interpreter,
	}
	fn, err := gen.Generate()
	if err != nil {
		return nil, fmt.Errorf("extract spell %s: %w", s.Command, err)
	}
	if err := MergeSpell(s, &fn); err != nil {
		return nil, fmt.Errorf("merge spell %s overrides: %w", s.Command, err)
	}
	return &fn, nil
}

// MergeSpell applies a scroll entry's overrides onto an extracted
// function: command name, source path, function name, interpreter, and
// value-only default overrides keyed by param name. The scroll can
// override a param's default, never its type — types come from source
// extraction alone.
func MergeSpell(s scroll.Spell, fn *Function) error {
	fn.CommandName = s.Command

	if s.Path != "" {
		fn.RelPathToSourceFile = s.Path
	}
	if s.Function != "" {
		fn.FunctionName = s.Function
	}
	for name, value := range s.Params {
		for i := range fn.Params {
			if fn.Params[i].Name == name {
				fn.Params[i].Default = value
				break
			}
		}
	}
	if s.Interpreter != "" {
		fn.Interpreter = s.Interpreter
	}
	return nil
}

func fromSteps(steps []scroll.Step) []Step {
	if steps == nil {
		return nil
	}
	out := make([]Step, len(steps))
	for i, s := range steps {
		out[i] = Step{
			Id:     s.Id,
			Spell:  s.Spell,
			Params: s.Params,

			If:   s.If,
			Then: fromSteps(s.Then),
			Else: fromSteps(s.Else),

			Let:   s.Let,
			Value: s.Value,

			Print: s.Print,
		}
	}
	return out
}
