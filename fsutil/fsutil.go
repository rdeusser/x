package fsutil

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyDir recursively copies a directory tree and attempts to preserve
// permissions.
func CopyDir(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		switch {
		case entry.IsDir():
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		case entry.Type()&os.ModeSymlink != 0:
			// skip symlinks for now
			continue
		default:
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// CopyFile copies a file from src to dst, preserving permissions if possible.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Open(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	// Force a sync.
	if err := out.Sync(); err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Copy permissions from original file.
	if err := os.Chmod(dst, info.Mode()); err != nil {
		return err
	}

	return nil
}
