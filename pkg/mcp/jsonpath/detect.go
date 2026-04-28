package jsonpath

import "github.com/ohler55/ojg/jp"

// IsRootLevelFilter returns true if the JSONPath expression starts with a filter
// at the root (e.g., $[?(@.field==value)]). Root-level filters require the input
// to be an array; when applied to a single object, ojg iterates the map's values
// instead of matching the object as a whole, so the filter silently returns nothing.
func IsRootLevelFilter(path string) bool {
	expr, err := jp.ParseString(path)
	if err != nil || len(expr) < 2 {
		return false
	}
	if _, ok := expr[0].(jp.Root); !ok {
		return false
	}
	_, ok := expr[1].(*jp.Filter)
	return ok
}
