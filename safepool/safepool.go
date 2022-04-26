package safepool

import "sync"

// A Pool is a type-safe wrapper around a sync.Pool.
type Pool[T any] struct {
	p *sync.Pool
	v T
}

// NewPool constructs a new Pool.
func NewPool[T any](t *T) Pool[T] {
	return Pool[T]{p: &sync.Pool{
		New: func() any {
			return t
		},
	}}
}

// Get retrieves T from the pool, creating one if necessary.
func (p Pool[T]) Get() *T {
	buf := p.p.Get().(*T)
	return buf
}

// Put adds t to the pool.
func (p Pool[T]) Put(t *T) {
	p.p.Put(t)
}
