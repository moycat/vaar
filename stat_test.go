package vaar

import (
	"os"
	"os/user"
	"strconv"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

var (
	me, _  = user.Current()
	grp, _ = user.LookupGroupId(me.Gid)
)

func TestStat(t *testing.T) {
	ts1 := time.Now()
	tmpDir := createStatTestFiles(t)
	ts2 := time.Now()
	e, err := Stat(tmpDir.Join("file1"))
	assert.NilError(t, err)
	t.Run("test regular file 1", testStatFile1(e, ts1, ts2))
	e, err = Stat(tmpDir.Join("file2"))
	assert.NilError(t, err)
	t.Run("test regular file 2", testStatFile2(e, ts1, ts2))
	e, err = Stat(tmpDir.Path())
	assert.NilError(t, err)
	t.Run("test directory", testStatDir(e, ts1, ts2))
	e, err = Stat(tmpDir.Join("link1"))
	assert.NilError(t, err)
	t.Run("test symlink", testStatSymlink(e, ts1, ts2))
}

func TestStatAt(t *testing.T) {
	ts1 := time.Now()
	tmpDir := createStatTestFiles(t)
	ts2 := time.Now()
	dir, err := os.Open(tmpDir.Path())
	assert.NilError(t, err)
	defer func() { _ = dir.Close() }()
	dirFd := int(dir.Fd())
	e, err := StatAt(dirFd, "file1")
	assert.NilError(t, err)
	t.Run("test regular file 1", testStatFile1(e, ts1, ts2))
	e, err = StatAt(dirFd, "file2")
	assert.NilError(t, err)
	t.Run("test regular file 2", testStatFile2(e, ts1, ts2))
	e, err = StatAt(dirFd, ".")
	assert.Equal(t, e.Name(), ".")
	e.name = "test" // Modify the name.
	assert.NilError(t, err)
	t.Run("test directory", testStatDir(e, ts1, ts2))
	e, err = StatAt(dirFd, "link1")
	assert.NilError(t, err)
	t.Run("test symlink", testStatSymlink(e, ts1, ts2))
}

func createStatTestFiles(t *testing.T) *fs.Dir {
	t.Helper()
	tmpDir := fs.NewDir(
		t, "test",
		fs.WithFile("file1", "7 bytes"),
		fs.WithFile("file2", "233", fs.WithMode(0o640)),
		fs.WithSymlink("link1", "file1"),
	)
	return tmpDir
}

func testStatFile1(e *Entry, ts1, ts2 time.Time) func(t *testing.T) {
	return func(t *testing.T) {
		assert.Equal(t, e.Name(), "file1")
		assert.Assert(t, e.Size() == 7)
		assert.Assert(t, !e.IsDir())
		assert.Equal(t, strconv.Itoa(int(e.OwnerID())), me.Uid)
		assert.Equal(t, strconv.Itoa(int(e.GroupID())), me.Gid)
		assert.Equal(t, e.Owner(), me.Username)
		assert.Equal(t, e.Group(), grp.Name)
		modTime := e.modTime.Round(time.Second)
		assert.Assert(t, !ts1.Round(time.Second).After(modTime) && !ts2.Round(time.Second).Before(modTime))
	}
}

func testStatFile2(e *Entry, ts1, ts2 time.Time) func(t *testing.T) {
	return func(t *testing.T) {
		assert.Equal(t, e.Name(), "file2")
		assert.Assert(t, e.Size() == 3)
		assert.Assert(t, e.Mode()&0o777 == 0o640)
		assert.Assert(t, !e.IsDir())
		assert.Equal(t, strconv.Itoa(int(e.OwnerID())), me.Uid)
		assert.Equal(t, strconv.Itoa(int(e.GroupID())), me.Gid)
		assert.Equal(t, e.Owner(), me.Username)
		assert.Equal(t, e.Group(), grp.Name)
		modTime := e.modTime.Round(time.Second)
		assert.Assert(t, !ts1.Round(time.Second).After(modTime) && !ts2.Round(time.Second).Before(modTime))
	}
}

func testStatDir(e *Entry, ts1, ts2 time.Time) func(t *testing.T) {
	return func(t *testing.T) {
		assert.Assert(t, strings.HasPrefix(e.Name(), "test"))
		assert.Assert(t, e.IsDir())
		assert.Equal(t, strconv.Itoa(int(e.OwnerID())), me.Uid)
		assert.Equal(t, strconv.Itoa(int(e.GroupID())), me.Gid)
		assert.Equal(t, e.Owner(), me.Username)
		assert.Equal(t, e.Group(), grp.Name)
		modTime := e.modTime.Round(time.Second)
		assert.Assert(t, !ts1.Round(time.Second).After(modTime) && !ts2.Round(time.Second).Before(modTime))
	}
}

func testStatSymlink(e *Entry, ts1, ts2 time.Time) func(t *testing.T) {
	return func(t *testing.T) {
		assert.Equal(t, e.Name(), "link1")
		assert.Assert(t, !e.IsDir())
		assert.Assert(t, strings.HasSuffix(e.Linkname(), "/file1"))
		assert.Equal(t, strconv.Itoa(int(e.OwnerID())), me.Uid)
		assert.Equal(t, strconv.Itoa(int(e.GroupID())), me.Gid)
		assert.Equal(t, e.Owner(), me.Username)
		assert.Equal(t, e.Group(), grp.Name)
		modTime := e.modTime.Round(time.Second)
		assert.Assert(t, !ts1.Round(time.Second).After(modTime) && !ts2.Round(time.Second).Before(modTime))
	}
}
