package set

import (
	"fmt"
	"strings"
	"sync"
)

type Set[T comparable] struct {
	m  map[T]struct{}
	mu sync.RWMutex
}

// Ensure Set satisfies set.Interface at compile-time.
var _ Interface[string] = (*Set[string])(nil)

// NewSet returns a set initialized with the provided items
func NewSet[T comparable](items ...T) Interface[T] {
	s := &Set[T]{
		m:  make(map[T]struct{}),
		mu: sync.RWMutex{},
	}

	for _, item := range items {
		s.Add(item)
	}

	return s
}

// Add an item to the set.
func (s *Set[T]) Add(item T) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	before := len(s.m)
	s.m[item] = struct{}{}

	return before != len(s.m)
}

// Remove an item from the set.
func (s *Set[T]) Remove(item T) bool {
	before := len(s.m)
	delete(s.m, item)

	return before != len(s.m)
}

// Clears removes all items from the set.
func (s *Set[T]) Clear() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.m = make(map[T]struct{})

	return len(s.m) == 0
}

// Contains determines whether the provided items are in the set.
func (s *Set[T]) Contains(items ...T) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, item := range items {
		if !s.contains(item) {
			return false
		}
	}

	return true
}

// Length returns the number of items in the set.
func (s *Set[T]) Length() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.m)
}

// ForEach iterates over items and executes the provided function against each
// item.
func (s *Set[T]) ForEach(fn func(T) bool) {
	for item := range s.m {
		if fn(item) {
			break
		}
	}
}

// String provides a string representation of the set.
func (s *Set[T]) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]string, 0, len(s.m))

	for item := range s.m {
		items = append(items, fmt.Sprint(item))
	}

	return fmt.Sprintf("Set{%s}", strings.Join(items, ", "))
}

// ToSlice returns the set as a slice.
func (s *Set[T]) ToSlice() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]T, 0, len(s.m))

	for item := range s.m {
		items = append(items, item)
	}

	return items
}

// IsSuperSet determines if every item in the provided set is in this set.
func (s *Set[T]) IsSuperSet(other Interface[T]) bool {
	o := other.(*Set[T])

	s.mu.RLock()
	defer s.mu.RUnlock()
	o.mu.RLock()
	defer o.mu.RUnlock()

	for item := range o.m {
		if !s.contains(item) {
			return false
		}
	}

	return true
}

// IsSubSet determines if every item in this set is in the provided set.
func (s *Set[T]) IsSubSet(other Interface[T]) bool {
	o := other.(*Set[T])

	s.mu.RLock()
	defer s.mu.RUnlock()
	o.mu.RLock()
	defer o.mu.RUnlock()

	for item := range s.m {
		if !o.contains(item) {
			return false
		}
	}

	return true
}

// Equal determines if the two sets are equal.
//
// Note: If both sets have the same number of items and contain the same
// items, they're equal. Order is irrelevant.
func (s *Set[T]) Equal(other Interface[T]) bool {
	o := other.(*Set[T])

	s.mu.RLock()
	defer s.mu.RUnlock()
	o.mu.RLock()
	defer o.mu.RUnlock()

	if s.Length() != o.Length() {
		return false
	}

	for item := range s.m {
		if !o.contains(item) {
			return false
		}
	}

	for item := range o.m {
		if !s.contains(item) {
			return false
		}
	}

	return true
}

// Intersect returns a new set containing only the items that exist in both
// sets.
func (s *Set[T]) Intersect(other Interface[T]) Interface[T] {
	o := other.(*Set[T])

	s.mu.RLock()
	defer s.mu.RUnlock()
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := NewSet[T]()

	// To eliminate looping over items of both sets, we can go over the biggest
	// set.
	if s.Length() > o.Length() {
		for item := range s.m {
			if o.contains(item) {
				result.Add(item)
			}
		}
	} else {
		for item := range o.m {
			if s.contains(item) {
				result.Add(item)
			}
		}
	}

	return result
}

// Difference returns a new set with items contained in this set that are not
// present in the provided set.
func (s *Set[T]) Difference(other Interface[T]) Interface[T] {
	o := other.(*Set[T])

	s.mu.RLock()
	defer s.mu.RUnlock()
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := NewSet[T]()

	for item := range s.m {
		if !o.contains(item) {
			result.Add(item)
		}
	}

	return result
}

// SymmetricDifference returns a new set with all items which are in either set,
// but not both.
func (s *Set[T]) SymmetricDifference(other Interface[T]) Interface[T] {
	o := other.(*Set[T])

	s.mu.RLock()
	defer s.mu.RUnlock()
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := NewSet[T]()

	for item := range s.m {
		if !o.contains(item) {
			result.Add(item)
		}
	}

	for item := range o.m {
		if !s.contains(item) {
			result.Add(item)
		}
	}

	return result
}

func (s *Set[T]) contains(item T) bool {
	_, ok := s.m[item]
	return ok
}
