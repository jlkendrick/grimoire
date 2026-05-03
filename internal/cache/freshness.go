package cache

import (
	descriptor "github.com/jlkendrick/grimoire/internal/descriptor"
)

func IsStale(descriptors *[]descriptor.FunctionDescriptor, scroll_path string) (bool, error) {
	for _, descriptor := range *descriptors {
		if descriptor.ScrollPath == scroll_path {
			return false, nil
		}
	}
	return true, nil
}