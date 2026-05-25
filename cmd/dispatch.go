package cmd

import (
	"fmt"
	"os"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
)

// loadedScroll bundles a parsed scroll with its reconciled descriptor cache
// so the two-pass resolver doesn't have to re-read the cache between passes.
type loadedScroll struct {
	scroll *scroll.Scroll
	cache  *cache.DescriptorCache
}

// registerScrollCommands resolves command names across all loaded scrolls
// and registers cobra commands on rootCmd. A command shared by 2+ scrolls,
// or matching a reserved static command, is registered as
// `<scroll-name>.<command>`; otherwise it stays bare. When two scrolls
// share the same Name and produce the same resolved command, the second
// is shadowed with a stderr warning.
func registerScrollCommands(scrolls []*scroll.Scroll) error {
	loaded := make([]loadedScroll, 0, len(scrolls))
	commandSources := map[string][]*scroll.Scroll{}

	for _, s := range scrolls {
		c, err := cache.ReadDescriptorCache(s.Path)
		if err != nil {
			return fmt.Errorf("loading cache for %s: %v", s.Path, err)
		}
		if err := resolve.ReconcileScrollAndDescriptors(s, c); err != nil {
			return err
		}
		if c.Functions == nil && c.Pipelines == nil {
			continue
		}
		loaded = append(loaded, loadedScroll{scroll: s, cache: c})
		for cmdName := range c.Functions {
			commandSources[cmdName] = append(commandSources[cmdName], s)
		}
		for cmdName := range c.Pipelines {
			commandSources[cmdName] = append(commandSources[cmdName], s)
		}
	}

	seen := map[string]string{}
	for _, ld := range loaded {
		resolvedNames := map[string]string{}
		for cmdName := range ld.cache.Functions {
			resolvedNames[cmdName] = resolveCommandName(ld.scroll, cmdName, commandSources)
		}
		for cmdName := range ld.cache.Pipelines {
			resolvedNames[cmdName] = resolveCommandName(ld.scroll, cmdName, commandSources)
		}

		cmds, err := GenerateCommands(ld.cache, resolvedNames, ld.scroll.Path)
		if err != nil {
			return fmt.Errorf("generating commands for %s: %v", ld.scroll.Path, err)
		}
		for _, c := range cmds {
			if prev, ok := seen[c.Use]; ok {
				fmt.Fprintf(os.Stderr, "warning: command %q in %s shadowed by earlier definition in %s\n", c.Use, ld.scroll.Path, prev)
				continue
			}
			seen[c.Use] = ld.scroll.Path
			rootCmd.AddCommand(c)
		}
	}
	return nil
}

// resolveCommandName decides whether a command should be registered bare or
// as a `<scroll-name>.<command>` dot-form. Dot-form applies when (a) two or
// more scrolls define the command, or (b) the command name matches a
// reserved static command (whose bare form would otherwise be unreachable
// because the static-command short-circuit at startup wins).
func resolveCommandName(s *scroll.Scroll, command string, sources map[string][]*scroll.Scroll) string {
	if staticCommands[command] || len(sources[command]) > 1 {
		return s.Name + "." + command
	}
	return command
}
