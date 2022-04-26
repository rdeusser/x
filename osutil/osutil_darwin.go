//go:build darwin

package osutil

func tempDir() string {
	return "/tmp"
}
