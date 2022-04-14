package errsync

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOnceReset(t *testing.T) {
	var (
		err    error
		once   Once
		called int
	)

	err = once.Do(func() error {
		called++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, called)

	err = once.Do(func() error {
		called++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, called)

	once.Reset()

	err = once.Do(func() error {
		called++
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, called)
}
