package vaar

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
	"gotest.tools/v3/fs"
)

func Test_readAhead(t *testing.T) {
	tmpFile := fs.NewFile(t, "test", fs.WithContent("test"))
	f, err := os.Open(tmpFile.Path())
	assert.NilError(t, err)
	defer func() { _ = f.Close() }()
	err = readAhead(int(f.Fd()), 128)
	assert.NilError(t, err)
}

func Test_chmodSymlink(t *testing.T) {
	tmpDir := fs.NewDir(
		t, "test",
		fs.WithFile("test", "test"),
		fs.WithSymlink("test_link", "test"),
	)
	err := chmodSymlink(tmpDir.Join("test_link"), 0o600)
	assert.NilError(t, err)
}
