package vaar

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// readAhead tells the kernel about reading a file in the near future, by issuing a F_RDAHEAD and a F_RDADVISE command.
func readAhead(fd, size int) error {
	_, err := unix.FcntlInt(uintptr(fd), unix.F_RDAHEAD, 1)
	if err != nil {
		return err
	}
	_, err = unix.FcntlInt(
		uintptr(fd),
		unix.F_RDADVISE,
		// unix.FcntlInt takes requires a pointer but takes an int, thus this ugly conversion.
		int(uintptr(unsafe.Pointer(&unix.Radvisory_t{
			Offset: 0,
			Count:  int32(size),
		}))),
	)
	return err
}

// chmodSymlink changes the permission of a symlink at path.
func chmodSymlink(path string, mode os.FileMode) error {
	return unix.Fchmodat(unix.AT_FDCWD, path, uint32(mode), unix.AT_SYMLINK_NOFOLLOW)
}
