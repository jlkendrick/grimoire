package graph

import (
	"fmt"
	"slices"

	ir "github.com/jlkendrick/grimoire/internal/ir"
)

// ValidatePipeline reports whether a pipeline is well-formed and would
// build, without resolving spells to runnable functions. It is the
// scroll-load entry point for ritual validation and composes the two
// halves of the answer:
//
//  1. a static pass over the IR — step shape rules, expression syntax,
//     and reference scoping (mode-aware: a pipe scope sees earlier
//     siblings and ancestors; a graph scope sees all siblings, because
//     there order is derived rather than declared)
//  2. the builder itself, run with a stub environment and its result
//     discarded — cycles, duplicate graph bindings, mode placement, and
//     every future structural rule come from the same code that builds
//     the runnable graph, so validation cannot drift from execution.
//
// spellExists confirms a spell name is known (descriptor cache and/or
// cross-scroll index); nil skips existence checks.
func ValidatePipeline(p *ir.Pipeline, spellExists func(name string) error) error {
	mode, err := effectiveMode(p.Mode, ir.ModePipe)
	if err != nil {
		return fmt.Errorf("pipeline %s: %w", p.Command, err)
	}
	if err := checkSteps(p.Steps, nil, mode, p.Command); err != nil {
		return err
	}

	env := Env{
		ResolveSpell: func(name string) (*ir.Function, error) {
			if spellExists != nil {
				if err := spellExists(name); err != nil {
					return nil, err
				}
			}
			return &ir.Function{CommandName: name}, nil
		},
	}
	_, err = BuildPipeline(p, env)
	return err
}

// checkSteps walks one scope enforcing shape rules, expression syntax,
// and reference visibility. inherited carries binding names from ancestor
// scopes — never from sibling branches; branch-internal names stay
// branch-internal.
func checkSteps(steps []ir.Step, inherited []string, mode ir.Mode, command string) error {
	visible := slices.Clone(inherited)
	if mode == ir.ModeGraph {
		// Graph scopes relax declaration order: any sibling binding is
		// referenceable, and the builder's cycle check catches abuse.
		for _, s := range steps {
			switch s.Kind() {
			case "spell":
				if s.Id != "" {
					visible = append(visible, s.Id)
				}
			case "let":
				visible = append(visible, s.Let)
			}
		}
	}

	for _, s := range steps {
		if err := checkShape(s, command); err != nil {
			return err
		}

		switch s.Kind() {
		case "spell":
			for _, v := range s.Params {
				// Only accessor-rooted values ("x.y", "x[0]") can be
				// statically judged: a bare string that names nothing
				// visible is a literal, not an error — the same rule the
				// runtime resolver applies.
				if root := accessorRoot(v); root != "" && !slices.Contains(visible, root) {
					return fmt.Errorf("pipeline %s: step %s references %s which is not in scope", command, s.Spell, root)
				}
			}
			if mode != ir.ModeGraph && s.Id != "" {
				visible = append(visible, s.Id)
			}

		case "let":
			if err := checkExprRefs(s.Value, visible, command, fmt.Sprintf("let %q", s.Let)); err != nil {
				return err
			}
			if mode != ir.ModeGraph {
				visible = append(visible, s.Let)
			}

		case "print":
			if err := checkExprRefs(s.Print, visible, command, "print"); err != nil {
				return err
			}

		case "if":
			if err := checkExprRefs(s.If, visible, command, "condition"); err != nil {
				return err
			}
			branchMode, err := effectiveMode(s.Mode, mode)
			if err != nil {
				return fmt.Errorf("pipeline %s: %w", command, err)
			}
			if err := checkSteps(s.Then, visible, branchMode, command); err != nil {
				return err
			}
			if err := checkSteps(s.Else, visible, branchMode, command); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkExprRefs(src string, visible []string, command, what string) error {
	roots, err := exprRoots(src)
	if err != nil {
		return fmt.Errorf("pipeline %s: invalid %s: %w", command, what, err)
	}
	for _, root := range roots {
		if !slices.Contains(visible, root) {
			return fmt.Errorf("pipeline %s: %s references %s which is not in scope", command, what, root)
		}
	}
	return nil
}

// checkShape enforces kind exclusivity. Kind() resolves mixed steps by
// precedence, so without these checks a step setting both spell: and if:
// would silently drop the spell.
func checkShape(s ir.Step, command string) error {
	switch s.Kind() {
	case "if":
		if s.Id != "" {
			return fmt.Errorf("pipeline %s: if-step cannot have an id", command)
		}
		if s.Spell != "" || s.Let != "" || s.Print != "" {
			return fmt.Errorf("pipeline %s: step cannot mix 'if' with spell/let/print fields", command)
		}
	case "let":
		if s.Spell != "" || s.Print != "" || len(s.Then) > 0 || len(s.Else) > 0 {
			return fmt.Errorf("pipeline %s: let-step %q cannot mix with spell/if/print fields", command, s.Let)
		}
		if s.Id != "" {
			return fmt.Errorf("pipeline %s: let-step uses 'let:' for the binding name; remove the redundant 'id:' field", command)
		}
		if s.Value == "" {
			return fmt.Errorf("pipeline %s: let %q requires a 'value:' expression", command, s.Let)
		}
	case "print":
		if s.Spell != "" || s.Id != "" || len(s.Params) > 0 || len(s.Then) > 0 || len(s.Else) > 0 || s.Value != "" {
			return fmt.Errorf("pipeline %s: print-step cannot mix with other step fields", command)
		}
	}
	return nil
}

// accessorRoot returns the root binding id of an accessor-path reference
// ("step.field", "step[0]") — empty for anything else. Bare strings are
// deliberately not treated as references here: statically they may be
// literals, and only the presence of an accessor makes reference intent
// unambiguous.
func accessorRoot(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	for i := 0; i < len(s); i++ {
		if s[i] == '.' || s[i] == '[' {
			return s[:i]
		}
	}
	return ""
}
