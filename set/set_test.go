package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdd(t *testing.T) {
	s := NewSet("foo", "bar")
	assert.Equal(t, 2, s.Length())
	s.Add("baz")
	assert.Equal(t, 3, s.Length())
}

func TestRemove(t *testing.T) {
	s := NewSet("foo", "bar", "baz")
	assert.Equal(t, 3, s.Length())
	s.Remove("baz")
	assert.Equal(t, 2, s.Length())
}

func TestClear(t *testing.T) {
	s := NewSet("foo", "bar", "baz")
	assert.Equal(t, 3, s.Length())
	s.Clear()
	assert.Equal(t, 0, s.Length())
}

func TestContains(t *testing.T) {
	s := NewSet("foo", "bar", "baz")
	assert.True(t, s.Contains("foo"))
	assert.True(t, s.Contains("foo", "bar"))
	assert.True(t, s.Contains("foo", "bar", "baz"))
}

func TestIsSuperSet(t *testing.T) {
	s := NewSet("foo", "bar", "baz")
	o := NewSet("foo")
	assert.True(t, s.IsSuperSet(o))
}

func TestIsSubSet(t *testing.T) {
	s := NewSet("foo")
	o := NewSet("foo", "bar", "baz")
	assert.True(t, s.IsSubSet(o))
}

func TestEqual(t *testing.T) {
	testCases := []struct {
		testName string
		s        Interface[string]
		o        Interface[string]
		want     bool
	}{
		{
			"not equal different length",
			NewSet("foo"),
			NewSet("foo", "bar", "baz"),
			false,
		},
		{
			"not equal same length",
			NewSet("foo", "bar", "qux"),
			NewSet("foo", "bar", "baz"),
			false,
		},
		{
			"equal",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz"),
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.Equal(tc.o))
		})
	}
}

func TestIntersect(t *testing.T) {
	testCases := []struct {
		testName string
		s        Interface[string]
		o        Interface[string]
		want     Interface[string]
	}{
		{
			"one item",
			NewSet("foo"),
			NewSet("foo", "bar", "baz"),
			NewSet("foo"),
		},
		{
			"two items",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "qux"),
			NewSet("foo", "bar"),
		},
		{
			"same items",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.Intersect(tc.o))
		})
	}
}

func TestDifference(t *testing.T) {
	testCases := []struct {
		testName string
		s        Interface[string]
		o        Interface[string]
		want     Interface[string]
	}{
		{
			"one item",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "qux"),
			NewSet("baz"),
		},
		{
			"two items",
			NewSet("foo", "bar", "baz", "qux", "quux"),
			NewSet("foo", "bar", "baz"),
			NewSet("qux", "quux"),
		},
		{
			"same items",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz"),
			NewSet[string](),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.Difference(tc.o))
		})
	}
}

func TestSymmetricDifference(t *testing.T) {
	testCases := []struct {
		testName string
		s        Interface[string]
		o        Interface[string]
		want     Interface[string]
	}{
		{
			"one item",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz", "qux"),
			NewSet("qux"),
		},
		{
			"two items",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz", "qux", "quux"),
			NewSet("qux", "quux"),
		},
		{
			"same items",
			NewSet("foo", "bar", "baz"),
			NewSet("foo", "bar", "baz"),
			NewSet[string](),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.s.SymmetricDifference(tc.o))
		})
	}
}
