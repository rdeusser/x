package set

type Interface[T comparable] interface {
	// Adds an item to the set.
	Add(T) bool

	// Removes an item from the set.
	Remove(T) bool

	// Removes all items from the set.
	Clear() bool

	// Returns whether the provided items are in the set.
	Contains(...T) bool

	// Returns the number of items in the set.
	Length() int

	// Iterates over items and executes the provided function against each
	// item.
	ForEach(func(T) bool)

	// Provides a string representation of the set.
	String() string

	// Returns the set as a slice.
	ToSlice() []T

	// Determines if every item in the provided set is in this set.
	IsSuperSet(Interface[T]) bool

	// Determines if every item in this set is in the provided set.
	IsSubSet(Interface[T]) bool

	// Determines if the two sets are equal.
	//
	// Note: If both sets have the same number of items and contain the same
	// items, they're equal. Order is irrelevant.
	Equal(Interface[T]) bool

	// Returns a new set containing only the items that exist in both sets.
	Intersect(Interface[T]) Interface[T]

	// Returns a new set with items contained in this set that are not present in
	// the provided set.
	Difference(Interface[T]) Interface[T]

	// Returns a new set with all items which are in either set, but not both.
	SymmetricDifference(Interface[T]) Interface[T]
}
