package vfs

import (
	"bytes"
	"io/fs"
	"path"
)

var _ Filesystem = &MemoryFs{}

type MemoryFs struct {
	files map[string]*MemoryFile
}

// Unlink implements Filesystem.
func (fs *MemoryFs) Unlink(filename string) error {
	return ErrNotImplemented
}

func NewMemoryFS(files map[string]*MemoryFile) *MemoryFs {
	return &MemoryFs{
		files: files,
	}
}

func (m *MemoryFs) Open(filename string) (File, error) {
	return getFile(m.files, filename)
}

func (fs *MemoryFs) ReadDir(path string) ([]fs.DirEntry, error) {
	return listDirFromFiles(fs.files, path)
}

// Stat implements Filesystem.
func (mfs *MemoryFs) Stat(filename string) (fs.FileInfo, error) {
	file, ok := mfs.files[filename]
	if !ok {
		return nil, ErrNotExist
	}
	return newFileInfo(path.Base(filename), file.Size()), nil
}

var _ File = &MemoryFile{}

type MemoryFile struct {
	name string
	*bytes.Reader
}

func NewMemoryFile(name string, data []byte) *MemoryFile {
	return &MemoryFile{
		name:   name,
		Reader: bytes.NewReader(data),
	}
}

func (d *MemoryFile) Stat() (fs.FileInfo, error) {
	return newFileInfo(d.name, int64(d.Reader.Len())), nil
}

func (d *MemoryFile) Size() int64 {
	return int64(d.Reader.Len())
}

func (d *MemoryFile) IsDir() bool {
	return false
}

func (d *MemoryFile) Close() (err error) {
	return
}
