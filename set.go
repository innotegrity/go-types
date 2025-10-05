package types

import "fmt"

// Set is a generic set of unique elements in any order.
//
// We map the object to a struct instead of a bool because empty structs take no space.
//
// If an element that already exists is added to the set, the newer element will be the one saved.
type Set[E comparable] map[E]struct{}

// NewSet creates a new Set object.
func NewSet[E comparable](vals ...E) Set[E] {
	s := Set[E]{}
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

// Add one or more values to the set.
func (s Set[E]) Add(vals ...E) {
	for _, v := range vals {
		s[v] = struct{}{}
	}
}

// Contains returns whether or not the set contains the given element.
func (s Set[E]) Contains(val E) bool {
	_, exists := s[val]
	return exists
}

// Intersection returns the overlapping elements in each set.
func (s Set[E]) Intersection(s2 Set[E]) Set[E] {
	result := NewSet[E]()
	for v := range s {
		if s2.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

// Members returns the elements in the set as a slice.
func (s Set[E]) Members() []E {
	members := make([]E, 0, len(s))
	for val := range s {
		members = append(members, val)
	}
	return members
}

// String returns the set formatted as a string.
func (s Set[E]) String() string {
	return fmt.Sprintf("%v", s.Members())
}

// Union returns a new set which is a union of the current set and the given set.
func (s Set[E]) Union(s2 Set[E]) Set[E] {
	result := NewSet(s.Members()...)
	result.Add(s2.Members()...)
	return result
}
