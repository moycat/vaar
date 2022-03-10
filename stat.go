package vaar

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

var (
	users  = make(map[uint32]string)
	groups = make(map[uint32]string)
)

// Stat stats a file and returns an Entry.
func Stat(path string) (*Entry, error) {
	var stat unix.Stat_t
	err := unix.Lstat(path, &stat)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}
	e := parseStat(path, &stat)
	if stat.Mode&unix.S_IFMT == unix.S_IFLNK {
		// Read the linkname if it's a symlink.
		e.linkname, err = readlink(path)
		if err != nil {
			return nil, fmt.Errorf("failed to readlink %s: %w", path, err)
		}
	}
	return e, nil
}

// StatAt stats a file in an opened directory and returns an Entry.
func StatAt(dirFd int, name string) (*Entry, error) {
	var stat unix.Stat_t
	err := unix.Fstatat(dirFd, name, &stat, unix.AT_SYMLINK_NOFOLLOW)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s at dir fd %d: %w", name, dirFd, err)
	}
	e := parseStat(name, &stat)
	if stat.Mode&unix.S_IFMT == unix.S_IFLNK {
		// Read the linkname if it's a symlink.
		// Also, use readlinkat with the already opened parent directory to save time.
		e.linkname, err = readlinkAt(dirFd, name)
		if err != nil {
			return nil, fmt.Errorf("failed to readlink %s at dir fd %d: %w", name, dirFd, err)
		}
	}
	return e, nil
}

// readlink wraps unix.Readlink and deal with some subtle situations.
func readlink(path string) (string, error) {
	for length := 256; ; length *= 2 {
		buf := make([]byte, length)
		var (
			n   int
			err error
		)
		for {
			n, err = unix.Readlink(path, buf)
			if err != syscall.EINTR {
				break
			}
		}
		if n <= 0 {
			n = 0
		}
		if err != nil {
			return "", err
		}
		if n < length {
			return string(buf[:n]), nil
		}
	}
}

// readlinkAt wraps unix.Readlinkat and deal with some subtle situations.
func readlinkAt(dirFd int, name string) (string, error) {
	for length := 256; ; length *= 2 {
		buf := make([]byte, length)
		var (
			n   int
			err error
		)
		for {
			n, err = unix.Readlinkat(dirFd, name, buf)
			if err != syscall.EINTR {
				break
			}
		}
		if n <= 0 {
			n = 0
		}
		if err != nil {
			return "", err
		}
		if n < length {
			return string(buf[:n]), nil
		}
	}
}

// parseStat parses a unix.Stat_t struct and returns an Entry.
func parseStat(name string, t *unix.Stat_t) *Entry {
	mode := os.FileMode(t.Mode & 0o777)
	switch t.Mode & unix.S_IFMT {
	case unix.S_IFBLK:
		mode |= os.ModeDevice
	case unix.S_IFCHR:
		mode |= os.ModeDevice | os.ModeCharDevice
	case unix.S_IFDIR:
		mode |= os.ModeDir
	case unix.S_IFIFO:
		mode |= os.ModeNamedPipe
	case unix.S_IFLNK:
		mode |= os.ModeSymlink
	case unix.S_IFSOCK:
		mode |= os.ModeSocket
	}
	if t.Mode&unix.S_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	if t.Mode&unix.S_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if t.Mode&unix.S_ISVTX != 0 {
		mode |= os.ModeSticky
	}
	modTime := time.Unix(t.Mtim.Sec, t.Mtim.Nsec)
	e := &Entry{
		name:    filepath.Base(name),
		size:    t.Size,
		mode:    mode,
		modTime: modTime,
		uid:     t.Uid,
		gid:     t.Gid,
		sys:     t,
	}
	// Parse owners.
	var ok bool
	if e.uname, ok = users[e.uid]; !ok {
		u, _ := user.LookupId(strconv.Itoa(int(e.uid)))
		if u != nil {
			users[e.uid] = u.Username
			e.uname = u.Username
		} else {
			users[e.uid] = ""
		}
	}
	if e.gname, ok = groups[e.gid]; !ok {
		g, _ := user.LookupGroupId(strconv.Itoa(int(e.gid)))
		if g != nil {
			groups[e.gid] = g.Name
			e.gname = g.Name
		} else {
			groups[e.gid] = ""
		}
	}
	return e
}
