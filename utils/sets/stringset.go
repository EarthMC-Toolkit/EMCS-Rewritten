package sets

// type SyncedStringSet struct {
// 	StringSet
// 	mu sync.Mutex
// }

type StringSet map[string]struct{}

// Return all elements in this set as a slice.
func (s StringSet) Keys() (keys []string) {
	for k := range s {
		keys = append(keys, k)
	}

	return
}

// Adds values to dst if they are not already present in this set.
//
// This is useful when an intermediate set is required to deduplicate X slice while adding elements from Y slice.
func (set StringSet) AppendSlice(dst *[]string, values []string) {
	for _, v := range values {
		if _, ok := set[v]; !ok {
			set[v] = struct{}{}
			*dst = append(*dst, v)
		}
	}
}
