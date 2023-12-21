package vfs

import (
	"io/fs"
	"os"
	"path"
	"sync"
)

type OsFS struct {
	hostDir string
}

// Stat implements Filesystem.
func (fs *OsFS) Stat(filename string) (fs.FileInfo, error) {
	if path.Clean(filename) == Separator {
		return newDirInfo(Separator), nil
	}

	return os.Stat(path.Join(fs.hostDir, filename))
}

// Open implements Filesystem.
func (fs *OsFS) Open(filename string) (File, error) {
	if path.Clean(filename) == Separator {
		return NewDir(filename), nil
	}

	osfile, err := os.Open(path.Join(fs.hostDir, filename))
	if err != nil {
		return nil, err
	}
	return NewOsFile(osfile), nil
}

// ReadDir implements Filesystem.
func (o *OsFS) ReadDir(dir string) ([]fs.DirEntry, error) {
	dir = path.Join(o.hostDir, dir)
	return os.ReadDir(dir)
}

func NewOsFs(osDir string) *OsFS {
	return &OsFS{
		hostDir: osDir,
	}
}

var _ Filesystem = &OsFS{}

type OsFile struct {
	f *os.File
}

func NewOsFile(f *os.File) *OsFile {
	return &OsFile{f: f}
}

var _ File = &OsFile{}

// Info implements File.
func (f *OsFile) Info() (fs.FileInfo, error) {
	return f.f.Stat()
}

// Close implements File.
func (f *OsFile) Close() error {
	return f.f.Close()
}

// Read implements File.
func (f *OsFile) Read(p []byte) (n int, err error) {
	return f.f.Read(p)
}

// ReadAt implements File.
func (f *OsFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.f.ReadAt(p, off)
}

func (f *OsFile) Stat() (fs.FileInfo, error) {
	return f.f.Stat()
}

// Size implements File.
func (f *OsFile) Size() int64 {
	stat, err := f.Stat()
	if err != nil {
		return 0
	}
	return stat.Size()
}

// IsDir implements File.
func (f *OsFile) IsDir() bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.IsDir()
}

type LazyOsFile struct {
	m    sync.Mutex
	path string
	file *os.File

	// cached field
	info fs.FileInfo
}

func NewLazyOsFile(path string) *LazyOsFile {
	return &LazyOsFile{path: path}
}

var _ File = &OsFile{}

func (f *LazyOsFile) open() error {
	f.m.Lock()
	defer f.m.Unlock()

	if f.file != nil {
		return nil
	}

	osFile, err := os.Open(f.path)
	if err != nil {
		return err
	}
	f.file = osFile
	return nil
}

// Close implements File.
func (f *LazyOsFile) Close() error {
	if f.file == nil {
		return nil
	}
	return f.file.Close()
}

// Read implements File.
func (f *LazyOsFile) Read(p []byte) (n int, err error) {
	err = f.open()
	if err != nil {
		return 0, err
	}
	return f.file.Read(p)
}

// ReadAt implements File.
func (f *LazyOsFile) ReadAt(p []byte, off int64) (n int, err error) {
	err = f.open()
	if err != nil {
		return 0, err
	}
	return f.file.ReadAt(p, off)
}

func (f *LazyOsFile) Stat() (fs.FileInfo, error) {
	f.m.Lock()
	if f.info == nil {
		if f.file == nil {
			info, err := os.Stat(f.path)
			if err != nil {
				return nil, err
			}
			f.info = info
		} else {
			info, err := f.file.Stat()
			if err != nil {
				return nil, err
			}
			f.info = info
		}
	}
	f.m.Unlock()
	return f.info, nil
}

// Size implements File.
func (f *LazyOsFile) Size() int64 {
	stat, err := f.Stat()
	if err != nil {
		return 0
	}
	return stat.Size()
}

// IsDir implements File.
func (f *LazyOsFile) IsDir() bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return stat.IsDir()
}
