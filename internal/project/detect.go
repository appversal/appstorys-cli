package project

import "fmt"

// Detect runs every adapter in restrictTo (or every supported adapter,
// if restrictTo is empty) against root and returns every platform that
// matched. It returns an empty, non-nil slice — not an error — when
// nothing matches; callers (the CLI layer) map that to exit code 3.
func Detect(root string, restrictTo []Platform) ([]ProjectInfo, error) {
	adapters := All()
	if len(restrictTo) > 0 {
		allowed := make(map[Platform]bool, len(restrictTo))
		for _, p := range restrictTo {
			allowed[p] = true
		}
		filtered := adapters[:0:0]
		for _, a := range adapters {
			if allowed[a.Platform()] {
				filtered = append(filtered, a)
			}
		}
		adapters = filtered
	}

	results := make([]ProjectInfo, 0)
	for _, a := range adapters {
		ok, info, err := a.Detect(root)
		if err != nil {
			return nil, fmt.Errorf("project: detecting %s: %w", a.Platform(), err)
		}
		if ok {
			results = append(results, info)
		}
	}
	return results, nil
}
