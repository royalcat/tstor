package nfs

import (
	"io/fs"
	"path/filepath"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/go-git/go-billy/v5"
)

type billyFsWrapper struct {
	fs vfs.Filesystem
}

var _ billy.Filesystem = (*billyFsWrapper)(nil)
var _ billy.Dir = (*billyFsWrapper)(nil)

// Chroot implements billy.Filesystem.
func (*billyFsWrapper) Chroot(path string) (billy.Filesystem, error) {
	return nil, billy.ErrNotSupported
}

// Create implements billy.Filesystem.
func (*billyFsWrapper) Create(filename string) (billy.File, error) {
	return nil, billy.ErrNotSupported
}

// Join implements billy.Filesystem.
func (*billyFsWrapper) Join(elem ...string) string {
	return filepath.Join(elem...)
}

// Lstat implements billy.Filesystem.
func (fs *billyFsWrapper) Lstat(filename string) (fs.FileInfo, error) {
	info, err := fs.fs.Stat(filename)
	if err != nil {
		return nil, billyErr(err)
	}
	return info, nil
}

// MkdirAll implements billy.Filesystem.
func (*billyFsWrapper) MkdirAll(filename string, perm fs.FileMode) error {
	return billy.ErrNotSupported
}

// Open implements billy.Filesystem.
func (f *billyFsWrapper) Open(filename string) (billy.File, error) {
	file, err := f.fs.Open(filename)
	if err != nil {
		return nil, billyErr(err)
	}
	return &billyFile{
		name: filename,
		file: file,
	}, nil
}

// OpenFile implements billy.Filesystem.
func (f *billyFsWrapper) OpenFile(filename string, flag int, perm fs.FileMode) (billy.File, error) {
	file, err := f.fs.Open(filename)
	if err != nil {
		return nil, billyErr(err)
	}
	return &billyFile{
		name: filename,
		file: file,
	}, nil
}

// ReadDir implements billy.Filesystem.
func (bfs *billyFsWrapper) ReadDir(path string) ([]fs.FileInfo, error) {
	ffs, err := bfs.fs.ReadDir(path)
	if err != nil {
		return nil, billyErr(err)
	}

	out := make([]fs.FileInfo, 0, len(ffs))
	for _, v := range ffs {
		if info, ok := v.(fs.FileInfo); ok {
			out = append(out, info)
		} else {
			info, err := v.Info()
			if err != nil {
				return nil, err
			}
			out = append(out, info)
		}

	}
	return out, nil
}

// Readlink implements billy.Filesystem.
func (*billyFsWrapper) Readlink(link string) (string, error) {
	return "", billy.ErrNotSupported
}

// Remove implements billy.Filesystem.
func (*billyFsWrapper) Remove(filename string) error {
	return billy.ErrNotSupported
}

// Rename implements billy.Filesystem.
func (*billyFsWrapper) Rename(oldpath string, newpath string) error {
	return billy.ErrNotSupported
}

// Root implements billy.Filesystem.
func (*billyFsWrapper) Root() string {
	return "/"
}

// Stat implements billy.Filesystem.
func (f *billyFsWrapper) Stat(filename string) (fs.FileInfo, error) {
	info, err := f.fs.Stat(filename)
	if err != nil {
		return nil, billyErr(err)
	}
	return info, nil
}

// Symlink implements billy.Filesystem.
func (*billyFsWrapper) Symlink(target string, link string) error {
	return billyErr(vfs.ErrNotImplemented)
}

// TempFile implements billy.Filesystem.
func (*billyFsWrapper) TempFile(dir string, prefix string) (billy.File, error) {
	return nil, billyErr(vfs.ErrNotImplemented)
}

type billyFile struct {
	name string
	file vfs.File
}

var _ billy.File = (*billyFile)(nil)

// Close implements billy.File.
func (f *billyFile) Close() error {
	return f.Close()
}

// Name implements billy.File.
func (f *billyFile) Name() string {
	return f.name
}

// Read implements billy.File.
func (f *billyFile) Read(p []byte) (n int, err error) {
	return f.Read(p)
}

// ReadAt implements billy.File.
func (f *billyFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.ReadAt(p, off)
}

// Seek implements billy.File.
func (*billyFile) Seek(offset int64, whence int) (int64, error) {
	return 0, billyErr(vfs.ErrNotImplemented)
}

// Truncate implements billy.File.
func (*billyFile) Truncate(size int64) error {
	return billyErr(vfs.ErrNotImplemented)
}

// Write implements billy.File.
func (*billyFile) Write(p []byte) (n int, err error) {
	return 0, billyErr(vfs.ErrNotImplemented)
}

// Lock implements billy.File.
func (*billyFile) Lock() error {
	return nil // TODO
}

// Unlock implements billy.File.
func (*billyFile) Unlock() error {
	return nil // TODO
}

func billyErr(err error) error {
	if err == vfs.ErrNotImplemented {
		return billy.ErrNotSupported
	}
	return err
}
