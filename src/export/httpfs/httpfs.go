package httpfs

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"sync"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"git.kmsign.ru/royalcat/tstor/src/iio"
)

var _ http.FileSystem = &HTTPFS{}

type HTTPFS struct {
	fs vfs.Filesystem
}

func NewHTTPFS(fs vfs.Filesystem) *HTTPFS {
	return &HTTPFS{fs: fs}
}

func (hfs *HTTPFS) Open(name string) (http.File, error) {
	f, err := hfs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	var fis []fs.FileInfo
	if f.IsDir() {
		// TODO make this lazy
		fis, err = hfs.filesToFileInfo(name)
		if err != nil {
			return nil, err
		}
	}

	return newHTTPFile(f, fis), nil
}

func (hfs *HTTPFS) filesToFileInfo(name string) ([]fs.FileInfo, error) {
	files, err := hfs.fs.ReadDir(name)
	if err != nil {
		return nil, err
	}

	out := make([]os.FileInfo, 0, len(files))
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			return nil, err
		}
		out = append(out, info)
	}

	return out, nil
}

var _ http.File = &httpFile{}

type httpFile struct {
	f vfs.File

	iio.ReaderSeeker

	mu sync.Mutex
	// dirPos is protected by mu.
	dirPos     int
	dirContent []os.FileInfo
}

func newHTTPFile(f vfs.File, dirContent []os.FileInfo) *httpFile {
	return &httpFile{
		f:            f,
		dirContent:   dirContent,
		ReaderSeeker: iio.NewSeekerWrapper(f, f.Size()),
	}
}

func (f *httpFile) Readdir(count int) ([]fs.FileInfo, error) {
	if !f.f.IsDir() {
		return nil, os.ErrInvalid
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	old := f.dirPos
	if old >= len(f.dirContent) {
		// The os.File Readdir docs say that at the end of a directory,
		// the error is io.EOF if count > 0 and nil if count <= 0.
		if count > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}
	if count > 0 {
		f.dirPos += count
		if f.dirPos > len(f.dirContent) {
			f.dirPos = len(f.dirContent)
		}
	} else {
		f.dirPos = len(f.dirContent)
		old = 0
	}

	return f.dirContent[old:f.dirPos], nil
}

func (f *httpFile) Stat() (fs.FileInfo, error) {
	return f.f.Stat()
}
