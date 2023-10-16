package vfs

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"sync"

	"git.kmsign.ru/royalcat/tstor/src/iio"
	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode/v2"
)

var ArchiveFactories = map[string]FsFactory{
	".zip": func(f File) (Filesystem, error) {
		return NewArchive(f, f.Size(), ZipLoader), nil
	},
	".rar": func(f File) (Filesystem, error) {
		return NewArchive(f, f.Size(), RarLoader), nil
	},
	".7z": func(f File) (Filesystem, error) {
		return NewArchive(f, f.Size(), SevenZipLoader), nil
	},
}

type ArchiveLoader func(r iio.Reader, size int64) (map[string]*archiveFile, error)

var _ Filesystem = &archive{}

type archive struct {
	r iio.Reader

	size int64

	files func() (map[string]*archiveFile, error)
}

func NewArchive(r iio.Reader, size int64, loader ArchiveLoader) *archive {
	return &archive{
		r:    r,
		size: size,
		files: sync.OnceValues(func() (map[string]*archiveFile, error) {
			return loader(r, size)
		}),
	}
}

func (a *archive) Open(filename string) (File, error) {
	files, err := a.files()
	if err != nil {
		return nil, err
	}

	return getFile(files, filename)
}

func (fs *archive) ReadDir(path string) (map[string]File, error) {
	files, err := fs.files()
	if err != nil {
		return nil, err
	}

	return listFilesInDir(files, path)
}

var _ File = &archiveFile{}

func NewArchiveFile(readerFunc func() (iio.Reader, error), len int64) *archiveFile {
	return &archiveFile{
		readerFunc: readerFunc,
		len:        len,
	}
}

type archiveFile struct {
	readerFunc func() (iio.Reader, error)
	reader     iio.Reader
	len        int64
}

func (d *archiveFile) load() error {
	if d.reader != nil {
		return nil
	}
	r, err := d.readerFunc()
	if err != nil {
		return err
	}

	d.reader = r

	return nil
}

func (d *archiveFile) Size() int64 {
	return d.len
}

func (d *archiveFile) IsDir() bool {
	return false
}

func (d *archiveFile) Close() (err error) {
	if d.reader != nil {
		err = d.reader.Close()
		d.reader = nil
	}

	return
}

func (d *archiveFile) Read(p []byte) (n int, err error) {
	if err := d.load(); err != nil {
		return 0, err
	}

	return d.reader.Read(p)
}

func (d *archiveFile) ReadAt(p []byte, off int64) (n int, err error) {
	if err := d.load(); err != nil {
		return 0, err
	}

	return d.reader.ReadAt(p, off)
}

var _ ArchiveLoader = ZipLoader

func ZipLoader(reader iio.Reader, size int64) (map[string]*archiveFile, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*archiveFile)
	for _, f := range zr.File {
		f := f
		if f.FileInfo().IsDir() {
			continue
		}

		rf := func() (iio.Reader, error) {
			zr, err := f.Open()
			if err != nil {
				return nil, err
			}

			return iio.NewDiskTeeReader(zr)
		}

		n := filepath.Join(string(os.PathSeparator), f.Name)
		af := NewArchiveFile(rf, f.FileInfo().Size())

		out[n] = af
	}

	return out, nil
}

var _ ArchiveLoader = SevenZipLoader

func SevenZipLoader(reader iio.Reader, size int64) (map[string]*archiveFile, error) {
	r, err := sevenzip.NewReader(reader, size)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*archiveFile)
	for _, f := range r.File {
		f := f
		if f.FileInfo().IsDir() {
			continue
		}

		rf := func() (iio.Reader, error) {
			zr, err := f.Open()
			if err != nil {
				return nil, err
			}

			return iio.NewDiskTeeReader(zr)
		}

		af := NewArchiveFile(rf, f.FileInfo().Size())
		n := filepath.Join(string(os.PathSeparator), f.Name)

		out[n] = af
	}

	return out, nil
}

var _ ArchiveLoader = RarLoader

func RarLoader(reader iio.Reader, size int64) (map[string]*archiveFile, error) {
	r, err := rardecode.NewReader(iio.NewSeekerWrapper(reader, size))
	if err != nil {
		return nil, err
	}

	out := make(map[string]*archiveFile)
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		rf := func() (iio.Reader, error) {
			return iio.NewDiskTeeReader(r)
		}

		n := filepath.Join(string(os.PathSeparator), header.Name)

		af := NewArchiveFile(rf, header.UnPackedSize)

		out[n] = af
	}

	return out, nil
}
