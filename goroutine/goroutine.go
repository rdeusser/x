package goroutine

// Launch creates a goroutine and returns an error.
func Launch(fn func() error) error {
	errc := make(chan error, 1)

	go func() {
		err := fn()
		errc <- err
	}()

	err := <-errc
	return err
}
