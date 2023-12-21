package vfs

import (
	"io/fs"
	"path"
)

var _ File = &dir{}

func NewDir(name string) File {
	return &dir{
		name: path.Base(name),
	}
}

type dir struct {
	name string
}

// Info implements File.
func (d *dir) Stat() (fs.FileInfo, error) {
	return newDirInfo(d.name), nil
}

func (d *dir) Size() int64 {
	return 0
}

func (d *dir) IsDir() bool {
	return true
}

func (d *dir) Close() error {
	return nil
}

func (d *dir) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (d *dir) ReadAt(p []byte, off int64) (n int, err error) {
	return 0, nil
}
