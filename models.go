package vaar

import (
	"os"
	"time"
)

type private interface{}

type Option func(i private) error

// dirent implements os.FileInfo.
type dirent struct {
	ino  uint64
	name string
	typ  byte
}

type entry struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	sys     interface{}
}

func (e *entry) Name() string {
	return e.name
}

func (e *entry) Size() int64 {
	return e.size
}

func (e *entry) Mode() os.FileMode {
	return e.mode
}

func (e *entry) ModTime() time.Time {
	return e.modTime
}

func (e *entry) IsDir() bool {
	return e.mode&os.ModeDir != 0
}

func (e *entry) Sys() interface{} {
	return e.sys
}
