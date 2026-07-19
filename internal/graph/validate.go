package graph

import (
	"fmt"
	"maps"
	"slices"

	expr "github.com/jlkendrick/grimoire/internal/expr"
	ir "github.com/jlkendrick/grimoire/internal/ir"
)

// ValidatePipeline reports whether a pipeline is well-formed and would
// build, without preparing anything to run. It is the scroll-load entry
// point for ritual validation and composes two halves:
//
//  1. a static pass over the IR — step shape rules, expression syntax,
//     reference scoping (mode-aware: a pipe scope sees earlier siblings
//     and ancestors; a graph scope sees all siblings, because there
//     order is derived rather than declared), and TYPE checking: a
//     static environment of binding name → type is threaded alongside
//     visibility, fed by spell return annotations, and every explicit
//     params: wire, condition, and let is checked against it. Typing is
//     gradual — Unknown passes everything, so unannotated scrolls are
//     never rejected.
//  2. the builder itself, run against a stub env with its result
//     discarded — cycles, duplicate graph bindings, mode placement, and
//     every future structural rule come from the same code that builds
//     the runnable graph, so validation cannot drift from execution.
//
// resolveSpell resolves a step's spell name to its Function (descriptor
// cache and/or cross-scroll index) so the type environment can see
// param and return annotations; nil skips existence and type checks.
func ValidatePipeline(p *ir.Pipeline, resolveSpell func(name string) (*ir.Function, error)) error {
	mode, err := effectiveMode(p.Mode, ir.ModePipe)
	if err != nil {
		return fmt.Errorf("pipeline %s: %w", p.Command, err)
	}
	if err := checkSteps(p.Steps, nil, nil, mode, p.Command, resolveSpell); err != nil {
		return err
	}

	env := Env{
		ResolveSpell: func(name string) (*ir.Function, error) {
			if resolveSpell != nil {
				return resolveSpell(name)
			}
			return &ir.Function{CommandName: name}, nil
		},
	}
	_, err = BuildPipeline(p, env)
	return err
}

// checkSteps walks one scope enforcing shape rules, expression syntax,
// reference visibility, and type compatibility. inherited/inheritedTypes
// carry binding names and types from ancestor scopes — never from
// sibling branches; branch-internal names stay branch-internal.
func checkSteps(steps []ir.Step, inherited []string, inheritedTypes map[string]*ir.TypeInfo, mode ir.Mode, command string, resolveSpell func(string) (*ir.Function, error)) error {
	visible := slices.Clone(inherited)
	types := maps.Clone(inheritedTypes)
	if types == nil {
		types = map[string]*ir.TypeInfo{}
	}

	// lookup is nil-safe and lenient: resolution failures surface in the
	// builder pass with proper errors; here they just mean Unknown.
	lookup := func(name string) *ir.Function {
		if resolveSpell == nil {
			return nil
		}
		fn, err := resolveSpell(name)
		if err != nil {
			return nil
		}
		return fn
	}

	if mode == ir.ModeGraph {
		// Graph scopes relax declaration order: any sibling binding is
		// referenceable (the builder's cycle check catches abuse), so
		// spell return types pre-bind for the whole scope. Let types
		// bind at their declaration position — a forward reference to a
		// let sees Unknown, which is lenient, not wrong.
		for _, s := range steps {
			switch s.Kind() {
			case "spell":
				if s.Id != "" {
					visible = append(visible, s.Id)
					if fn := lookup(s.Spell); fn != nil {
						types[s.Id] = fn.Returns
					}
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
			fn := lookup(s.Spell)
			for name, v := range s.Params {
				// Only accessor-rooted values ("x.y", "x[0]") can be
				// statically judged for scope: a bare string that names
				// nothing visible is a literal, not an error — the same
				// rule the runtime resolver applies.
				if root := accessorRoot(v); root != "" && !slices.Contains(visible, root) {
					return fmt.Errorf("pipeline %s: step %s references %s which is not in scope", command, s.Spell, root)
				}
				if fn == nil {
					continue
				}
				expected := paramType(fn, name)
				actual, err := paramValueType(v, visible, types)
				if err != nil {
					return fmt.Errorf("pipeline %s: step %s param %s: %w", command, s.Spell, name, err)
				}
				if err := ir.Compatible(expected, actual); err != nil {
					return fmt.Errorf("pipeline %s: step %s param %s: %w", command, s.Spell, name, err)
				}
			}
			if mode != ir.ModeGraph && s.Id != "" {
				visible = append(visible, s.Id)
				if fn != nil {
					types[s.Id] = fn.Returns
				}
			}

		case "let":
			if err := checkExprRefs(s.Value, visible, command, fmt.Sprintf("let %q", s.Let)); err != nil {
				return err
			}
			t, err := inferType(s.Value, types)
			if err != nil {
				return fmt.Errorf("pipeline %s: let %q: %w", command, s.Let, err)
			}
			types[s.Let] = t
			if mode != ir.ModeGraph {
				visible = append(visible, s.Let)
			}

		case "print":
			if err := checkExprRefs(s.Print, visible, command, "print"); err != nil {
				return err
			}
			if _, err := inferType(s.Print, types); err != nil {
				return fmt.Errorf("pipeline %s: print: %w", command, err)
			}

		case "if":
			if err := checkExprRefs(s.If, visible, command, "condition"); err != nil {
				return err
			}
			t, err := inferType(s.If, types)
			if err != nil {
				return fmt.Errorf("pipeline %s: condition %q: %w", command, s.If, err)
			}
			// The strict-bool contract, enforced statically when known.
			if err := ir.RequireBool(t); err != nil {
				return fmt.Errorf("pipeline %s: condition %q: %w", command, s.If, err)
			}
			branchMode, err := effectiveMode(s.Mode, mode)
			if err != nil {
				return fmt.Errorf("pipeline %s: %w", command, err)
			}
			if err := checkSteps(s.Then, visible, types, branchMode, command, resolveSpell); err != nil {
				return err
			}
			if err := checkSteps(s.Else, visible, types, branchMode, command, resolveSpell); err != nil {
				return err
			}
		}
	}
	return nil
}

// paramValueType types one params: value. A string that names a visible
// binding (bare or accessor-rooted) is a reference and walks the type
// environment; any other value is a YAML literal and types as itself.
func paramValueType(v any, visible []string, types map[string]*ir.TypeInfo) (*ir.TypeInfo, error) {
	if s, ok := v.(string); ok {
		root := expr.ReferenceRoot(s)
		if slices.Contains(visible, root) {
			return ir.WalkRef(types[root], s[len(root):])
		}
		if accessorRoot(v) != "" {
			// Accessor-rooted but out of scope: the scope check already
			// rejected it; nothing further to say about its type.
			return nil, nil
		}
	}
	return ir.TypeOfLiteral(v), nil
}

// paramType returns the declared type of fn's parameter, or nil for a
// name the signature doesn't declare — which is not an error here: the
// function may take **kwargs or the extractor may have skipped the
// parameter, and Unknown is the honest answer.
func paramType(fn *ir.Function, name string) *ir.TypeInfo {
	for _, p := range fn.Params {
		if p.Name == name {
			return p.ResolvedType
		}
	}
	return nil
}

func inferType(src string, types map[string]*ir.TypeInfo) (*ir.TypeInfo, error) {
	parsed, err := expr.ParseCondition(src)
	if err != nil {
		// Parse errors are reported (with better context) by
		// checkExprRefs; treat as Unknown here.
		return nil, nil
	}
	return ir.InferExpr(parsed, types)
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
