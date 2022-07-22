package set

// Set implements a Set, using the go map as the underlying storage.
type Set[K comparable] struct {
	storage map[K]struct{}
}

// New returns an empty Set.
func New[K comparable]() Set[K] {
	return Set[K]{
		storage: make(map[K]struct{}),
	}
}

// Of returns a new Set initialized with the given 'vals'
func Of[K comparable](vals ...K) Set[K] {
	s := New[K]()
	for _, val := range vals {
		s.Put(val)
	}
	return s
}

// Put adds 'val' to the set.
func (s Set[K]) Put(val K) {
	s.storage[val] = struct{}{}
}

// Has returns true only if 'val' is in the set.
func (s Set[K]) Has(val K) bool {
	_, ok := s.storage[val]
	return ok
}

// Remove removes 'val' from the set.
func (s Set[K]) Remove(val K) {
	delete(s.storage, val)
}

// ToArray returns go slice
func (s Set[K]) ToArray() (arr []K) {
	for v := range s.storage {
		arr = append(arr, v)
	}
	return arr
}
