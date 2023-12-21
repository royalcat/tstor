package vfs

import (
	"errors"
	"io/fs"
	"path"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/iio"
)

type File interface {
	IsDir() bool
	Size() int64
	Stat() (fs.FileInfo, error)

	iio.Reader
}

var ErrNotImplemented = errors.New("not implemented")

type Filesystem interface {
	// Open opens the named file for reading. If successful, methods on the
	// returned file can be used for reading; the associated file descriptor has
	// mode O_RDONLY.
	Open(filename string) (File, error)

	// ReadDir reads the directory named by dirname and returns a list of
	// directory entries.
	ReadDir(path string) ([]fs.DirEntry, error)

	Stat(filename string) (fs.FileInfo, error)
}

const defaultMode = fs.FileMode(0555)

type fileInfo struct {
	name  string
	size  int64
	isDir bool
}

var _ fs.FileInfo = &fileInfo{}
var _ fs.DirEntry = &fileInfo{}

func newDirInfo(name string) *fileInfo {
	return &fileInfo{
		name:  path.Base(name),
		size:  0,
		isDir: true,
	}
}

func newFileInfo(name string, size int64) *fileInfo {
	return &fileInfo{
		name:  path.Base(name),
		size:  size,
		isDir: false,
	}
}

func (fi *fileInfo) Info() (fs.FileInfo, error) {
	return fi, nil
}

func (fi *fileInfo) Type() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir
	}

	return 0
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return fi.size
}

func (fi *fileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return defaultMode | fs.ModeDir
	}

	return defaultMode
}

func (fi *fileInfo) ModTime() time.Time {
	// TODO fix it
	return time.Time{}
}

func (fi *fileInfo) IsDir() bool {
	return fi.isDir
}

func (fi *fileInfo) Sys() interface{} {
	return nil
}
