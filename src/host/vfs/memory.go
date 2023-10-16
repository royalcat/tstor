package vfs

import (
	"bytes"
)

var _ Filesystem = &MemoryFs{}

type MemoryFs struct {
	files map[string]*MemoryFile
}

func NewMemoryFS(files map[string]*MemoryFile) *MemoryFs {
	return &MemoryFs{
		files: files,
	}
}

func (m *MemoryFs) Open(filename string) (File, error) {
	return getFile(m.files, filename)
}

func (fs *MemoryFs) ReadDir(path string) (map[string]File, error) {
	return listFilesInDir(fs.files, path)
}

var _ File = &MemoryFile{}

type MemoryFile struct {
	*bytes.Reader
}

func NewMemoryFile(data []byte) *MemoryFile {
	return &MemoryFile{
		Reader: bytes.NewReader(data),
	}
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
