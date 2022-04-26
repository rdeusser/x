//go:build aix && android && dragonfly && freebsd && illumos && ios && js && linux && netbsd && openbsd && plan9 && solaris && windows

package osutil

import "os"

func tempDir() string {
	return os.TempDir()
}
