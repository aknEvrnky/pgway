package schema

import "slices"

// applyKindRank defines bottom-up apply order for multi-document YAML.
var applyKindRank = map[string]int{
	"Proxy":        0,
	"Pool":         1,
	"LoadBalancer": 2,
	"Router":       3,
	"Flow":         4,
	"Entrypoint":   5,
}

// SortByApplyOrder returns resources sorted by kind dependency order
// (proxy → pool → balancer → router → flow → entrypoint). Resources of the
// same kind keep their relative order. Unknown kinds sort after known kinds,
// preserving relative order among themselves.
func SortByApplyOrder(resources []RawResource) []RawResource {
	if len(resources) < 2 {
		return resources
	}
	out := slices.Clone(resources)
	slices.SortStableFunc(out, func(a, b RawResource) int {
		ra, okA := applyKindRank[a.Kind]
		rb, okB := applyKindRank[b.Kind]
		if !okA {
			ra = len(applyKindRank)
		}
		if !okB {
			rb = len(applyKindRank)
		}
		return ra - rb
	})
	return out
}
