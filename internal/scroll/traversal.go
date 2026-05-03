package scroll

import (
	"errors"
	
	utils "github.com/jlkendrick/grimoire/internal/utils"
)

// ErrScrollExists is returned by InitScroll when a scroll.yaml already exists
// in the target directory.
var ErrScrollExists = errors.New("scroll already exists")

// FindLocalScroll walks upward from start_dir looking for scroll.yaml. Returns
// the absolute path and true on hit, ("", false) otherwise.
func FindLocalScroll(start_dir string) (string, bool) {
	matched, found := utils.UpwardsTraversalForTargets(start_dir, []string{"scroll.yaml"})
	if !found {
		return "", false
	}
	return matched["scroll.yaml"], true
}