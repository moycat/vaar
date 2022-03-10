package vaar

import (
	"os"

	"golang.org/x/sys/unix"
)

// readAhead tells the kernel about reading a file in the near future, by issuing a fadvise64 syscall.
func readAhead(fd, size int) error {
	return unix.Fadvise(fd, 0, int64(size), unix.FADV_SEQUENTIAL)
}

// chmodSymlink does nothing, as Linux doesn't support file modes of symlinks.
func chmodSymlink(_ string, _ os.FileMode) error {
	return nil
}
