package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	cache "github.com/jlkendrick/grimoire/internal/cache"
	resolve "github.com/jlkendrick/grimoire/internal/resolve"
	scroll "github.com/jlkendrick/grimoire/internal/scroll"
	weave "github.com/jlkendrick/grimoire/weave"
)

var weave_cmd = &cobra.Command{
	Use:   "weave [path_to_file.wv]",
	Short: "Transpile a .wv ritual into the local scroll.yaml",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input_path := args[0]
		if !strings.HasSuffix(input_path, ".wv") {
			fmt.Printf("Error: expected a .wv file, got %s\n", input_path)
			return
		}
		abs_input_path, err := filepath.Abs(input_path)
		if err != nil {
			fmt.Printf("Error resolving input path: %v\n", err)
			return
		}

		// 1. Parse + transpile the .wv into a scroll.Ritual
		ritual, err := weave.WeaveFile(abs_input_path)
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		// 2. Locate (or initialize) the local scroll we'll write into
		scroll_obj, err := scroll.ResolveAddScroll()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		// 3. Bring the descriptor cache for that scroll up-to-date so the
		//    validation below sees all spells currently in the scroll. We
		//    also reconcile every other registered scroll's spells so a
		//    ritual that uses cross-scroll dot-notation refs (e.g.
		//    `other.do_thing`) can be validated against the full set.
		descriptor_cache, err := cache.ReadDescriptorCache(scroll_obj.Path)
		if err != nil {
			fmt.Printf("Error loading descriptor cache: %v\n", err)
			return
		}
		if err := resolve.ReconcileSpellsOnly(scroll_obj, descriptor_cache); err != nil {
			fmt.Printf("%v\n", err)
			return
		}

		spellIndex, _, err := buildGlobalSpellIndex()
		if err != nil {
			fmt.Printf("Warning: could not build cross-scroll index (cross-scroll refs will not resolve): %v\n", err)
			spellIndex = nil
		}

		// 4. In-memory validation: same checks the pipeline reconciler runs
		//    when it materializes a ritual descriptor (first step is a spell,
		//    referenced spells exist locally or via the spell index,
		//    step-id refs are in scope, condition expressions parse).
		if err := resolve.ValidateRitual(ritual, descriptor_cache, spellIndex); err != nil {
			fmt.Printf("Error validating ritual: %v\n", err)
			return
		}

		// 5. Append and write. The next CLI invocation will reconcile the new
		//    ritual into the descriptor cache automatically.
		scroll_obj.Rituals = append(scroll_obj.Rituals, ritual)
		if err := scroll_obj.Write(); err != nil {
			fmt.Printf("Error writing scroll: %v\n", err)
			return
		}

		fmt.Printf("%s Woven ritual %s into scroll %s\n",
			accent_style("+"),
			spell_style(ritual.Command),
			dim_style(scroll_obj.Path),
		)
	},
}

func init() {
	rootCmd.AddCommand(weave_cmd)
}
