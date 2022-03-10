package vaar

import (
	"os"
	"time"
)

type private interface{}

type Option func(i private) error

type dirent struct {
	ino  uint64
	name string
	typ  byte
}

// Entry represents a file, used in WalkFunc, specialized for tar headers.
type Entry struct {
	name     string
	size     int64
	mode     os.FileMode
	modTime  time.Time
	sys      interface{}
	linkname string
	uid      uint32
	gid      uint32
	uname    string
	gname    string
}

func (e *Entry) Name() string {
	return e.name
}

func (e *Entry) Size() int64 {
	return e.size
}

func (e *Entry) Mode() os.FileMode {
	return e.mode
}

func (e *Entry) ModTime() time.Time {
	return e.modTime
}

func (e *Entry) IsDir() bool {
	return e.mode&os.ModeDir != 0
}

func (e *Entry) Sys() interface{} {
	return nil
}

func (e *Entry) Linkname() string {
	return e.linkname
}

func (e *Entry) OwnerID() uint32 {
	return e.uid
}

func (e *Entry) GroupID() uint32 {
	return e.gid
}

func (e *Entry) Owner() string {
	return e.uname
}

func (e *Entry) Group() string {
	return e.gname
}
