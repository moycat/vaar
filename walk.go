package vaar

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const (
	dentBufSize = 8 << 20   // 8 MiB
	adviceSize  = 256 << 10 // 256 KiB
)

// WalkFunc is the type of the function called by Walk to visit each file or directory.
// The path argument is the absolute path of the current file.
// The linkName argument is the link target if this file is a symlink, otherwise empty.
// The info argument is a fs.FileInfo struct about this file.
// The r argument is the reader of this file, if it's a regular one. It must be closed if it's not nil.
type WalkFunc func(path, linkName string, info fs.FileInfo, r io.ReadCloser) error

// Walk is a walking function similar to filepath.Walk, but differs in these aspects:
//   1. It takes WalkFunc instead of filepath.WalkFunc.
//   2. The passed-in path must be a directory.
//   3. The error that occurred during walking is directly returned, without passing to WalkFunc.
//   4. Lots of magic targeting *nix systems. See the comments for details.
func Walk(path string, walkFunc WalkFunc) error {
	// Memory allocations are expensive. Use a pool to reuse buffers.
	dentBufPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, dentBufSize)
		},
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return errors.WithMessage(err, "failed to calculate the absolute path")
	}
	// As we only need the file descriptors afterwards, we directly use the syscall.
	// We should avoid os.OpenFile, because an os.File is closed by Go runtime on GC.
	dirFd, err := unix.Open(path, os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return errors.WithMessage(err, "failed to open the walk path")
	}
	// The root path needs manual walk.
	var stat unix.Stat_t
	if err := unix.Fstat(dirFd, &stat); err != nil {
		return errors.WithMessage(err, "failed to stat the walk path")
	}
	if err := walkFunc(path, "", parseStat(filepath.Base(path), &stat), nil); err != nil {
		return err
	}
	return walk(path, dirFd, dentBufPool, walkFunc)
}

// walk does the real walking stuff. It receives a opened directory and iterates the items in it.
func walk(dirName string, dirFd int, dentBufPool *sync.Pool, walkFunc WalkFunc) error {
	buf := dentBufPool.Get().([]byte)
	defer func() {
		// The directory is closed when this function ends, and the dent buffer is returned to the pool.
		_ = unix.Close(dirFd)
		dentBufPool.Put(buf)
	}()
	for {
		// Use a large buffer to get directory entries.
		// The buffer size in os.ReadDir is 8 KiB and is too small, causing many unnecessary syscall operations.
		n, err := unix.ReadDirent(dirFd, buf)
		if err != nil {
			return errors.WithMessagef(err, "failed to call getdents64 on %s", dirName)
		}
		if n == 0 {
			return nil
		}
		dirents := parseDirentBuf(buf[:n])
		for _, dent := range dirents {
			// Use fstatat with the fd of the already opened parent directory to save time.
			// If we use the full path directly, the kernel has to walk through the full path and do heavy checks
			// like the permission.
			var stat unix.Stat_t
			err := unix.Fstatat(dirFd, dent.name, &stat, unix.AT_SYMLINK_NOFOLLOW)
			if err != nil {
				return errors.WithMessagef(err, "failed to stat %s in %s", dent.name, dirName)
			}
			fileInfo := parseStat(dent.name, &stat)
			filePath := filepath.Join(dirName, dent.name)
			var (
				reader   io.ReadCloser
				linkName string
			)
			switch dent.typ {
			case unix.DT_REG:
				// A regular file should be opened for reading.
				// Also, use openat with the the already opened parent directory to save time.
				fd, err := unix.Openat(dirFd, dent.name, os.O_RDONLY, 0)
				if err != nil {
					return errors.WithMessagef(err, "failed to open %s for reading", filePath)
				}
				reader = os.NewFile(uintptr(fd), dent.name)
				// Issue a read ahead instruction to the kernel to prefetch the file content.
				_ = readAhead(fd, adviceSize)
			case unix.DT_LNK:
				// A symlink needs its target.
				// Also, use readlinkat with the already opened parent directory to save time.
				linkName, err = readlinkAt(dirFd, dent.name)
				if err != nil {
					return errors.WithMessagef(err, "failed to read link name of %s", filePath)
				}
			}
			if err := walkFunc(filePath, linkName, fileInfo, reader); err != nil {
				return err
			}
			if fileInfo.IsDir() {
				// Walk the sub-directories recursively.
				// Also, use openat with the already opened parent directory to save time.
				nextDirFd, err := unix.Openat(dirFd, dent.name, unix.O_RDONLY|unix.O_DIRECTORY, 0)
				if err != nil {
					return errors.WithMessagef(err, "failed to open directory %s in %s", dent.name, dirName)
				}
				if err := walk(filePath, nextDirFd, dentBufPool, walkFunc); err != nil {
					return err
				}
			}
		}
	}
}

// parseDirentBuf parses the dir entries returned by the syscall.
func parseDirentBuf(buf []byte) []*dirent {
	dirents := make([]*dirent, 0, len(buf)>>5) // Divided by 32, a reasonable guess to avoid capacity growth.
	for len(buf) > 0 {
		unixDirent := (*unix.Dirent)(unsafe.Pointer(&buf[0]))
		buf = buf[unixDirent.Reclen:]
		name := getDirentName(unixDirent)
		if name == "." || name == ".." {
			// Ignore the useless parent entries.
			continue
		}
		dent := &dirent{
			ino:  unixDirent.Ino,
			name: name,
			typ:  unixDirent.Type,
		}
		dirents = append(dirents, dent)
	}
	// Reading files in ascending inode order significantly improves read performance when page cache is absent.
	// It applies to EXT4 and XFS at least. Other filesystems haven't been tested yet.
	// When the length is less than 3, we skip sort.Slice to avoid reflection overheads.
	switch len(dirents) {
	case 0, 1:
	case 2:
		if dirents[0].ino > dirents[1].ino {
			dirents[0], dirents[1] = dirents[1], dirents[0]
		}
	default:
		sort.Slice(dirents, func(i, j int) bool {
			return dirents[i].ino < dirents[j].ino
		})
	}
	return dirents
}

// getDirentName returns the name field of a unix.Dirent.
func getDirentName(dirent *unix.Dirent) string {
	name := make([]byte, len(dirent.Name))
	var i int
	for ; i < len(dirent.Name); i++ {
		if dirent.Name[i] == 0 {
			break
		}
		name[i] = byte(dirent.Name[i])
	}
	return string(name[:i])
}

// parseStat parses a unix.Stat_t struct and returns an os.FileInfo.
func parseStat(name string, t *unix.Stat_t) os.FileInfo {
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
	return &entry{name: name, size: t.Size, mode: mode, modTime: modTime}
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
