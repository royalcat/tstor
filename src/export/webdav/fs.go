package webdav

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"git.kmsign.ru/royalcat/tstor/src/iio"
	"golang.org/x/net/webdav"
)

var _ webdav.FileSystem = &WebDAV{}

type WebDAV struct {
	fs vfs.Filesystem
}

func newFS(fs vfs.Filesystem) *WebDAV {
	return &WebDAV{fs: fs}
}

func (wd *WebDAV) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	name = vfs.AbsPath(name)

	// TODO handle flag and permissions
	f, err := wd.lookupFile(name)
	if err != nil {
		return nil, err
	}

	wdf := newFile(path.Base(name), f, func() ([]fs.FileInfo, error) {
		return wd.listDir(name)
	})
	return wdf, nil
}

func (wd *WebDAV) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	return wd.fs.Stat(vfs.AbsPath(name))
}

func (wd *WebDAV) Mkdir(ctx context.Context, name string, perm fs.FileMode) error {
	return webdav.ErrNotImplemented
}

func (wd *WebDAV) RemoveAll(ctx context.Context, name string) error {
	return wd.fs.Unlink(name)
}

func (wd *WebDAV) Rename(ctx context.Context, oldName, newName string) error {
	return webdav.ErrNotImplemented
}

func (wd *WebDAV) lookupFile(name string) (vfs.File, error) {
	return wd.fs.Open(path.Clean(name))
}

func (wd *WebDAV) listDir(path string) ([]os.FileInfo, error) {
	files, err := wd.fs.ReadDir(path)
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

var _ webdav.File = &webDAVFile{}

type webDAVFile struct {
	iio.Reader

	fi os.FileInfo

	mudp   sync.Mutex
	dirPos int

	mup        sync.Mutex
	pos        int64
	dirFunc    func() ([]os.FileInfo, error)
	dirContent []os.FileInfo
}

func newFile(name string, f vfs.File, df func() ([]os.FileInfo, error)) *webDAVFile {
	return &webDAVFile{
		fi:      newFileInfo(name, f.Size(), f.IsDir()),
		dirFunc: df,
		Reader:  f,
	}
}

func (wdf *webDAVFile) Readdir(count int) ([]os.FileInfo, error) {
	wdf.mudp.Lock()
	defer wdf.mudp.Unlock()

	if !wdf.fi.IsDir() {
		return nil, os.ErrInvalid
	}

	if wdf.dirContent == nil {
		dc, err := wdf.dirFunc()
		if err != nil {
			return nil, err
		}
		wdf.dirContent = dc
	}

	old := wdf.dirPos
	if old >= len(wdf.dirContent) {
		// The os.File Readdir docs say that at the end of a directory,
		// the error is io.EOF if count > 0 and nil if count <= 0.
		if count > 0 {
			return nil, io.EOF
		}
		return nil, nil
	}
	if count > 0 {
		wdf.dirPos += count
		if wdf.dirPos > len(wdf.dirContent) {
			wdf.dirPos = len(wdf.dirContent)
		}
	} else {
		wdf.dirPos = len(wdf.dirContent)
		old = 0
	}

	return wdf.dirContent[old:wdf.dirPos], nil
}

func (wdf *webDAVFile) Stat() (os.FileInfo, error) {
	return wdf.fi, nil
}

func (wdf *webDAVFile) Read(p []byte) (int, error) {
	wdf.mup.Lock()
	defer wdf.mup.Unlock()

	n, err := wdf.Reader.ReadAt(p, wdf.pos)
	wdf.pos += int64(n)

	return n, err
}

func (wdf *webDAVFile) Seek(offset int64, whence int) (int64, error) {
	wdf.mup.Lock()
	defer wdf.mup.Unlock()

	switch whence {
	case io.SeekStart:
		wdf.pos = offset
	case io.SeekCurrent:
		wdf.pos = wdf.pos + offset
	case io.SeekEnd:
		wdf.pos = wdf.fi.Size() + offset
	}

	return wdf.pos, nil
}

func (wdf *webDAVFile) Write(p []byte) (n int, err error) {
	return 0, webdav.ErrNotImplemented
}

type webDAVFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func newFileInfo(name string, size int64, isDir bool) *webDAVFileInfo {
	return &webDAVFileInfo{
		name:  name,
		size:  size,
		isDir: isDir,
	}
}

func (wdfi *webDAVFileInfo) Name() string {
	return wdfi.name
}

func (wdfi *webDAVFileInfo) Size() int64 {
	return wdfi.size
}

func (wdfi *webDAVFileInfo) Mode() os.FileMode {
	if wdfi.isDir {
		return 0555 | os.ModeDir
	}

	return 0555
}

func (wdfi *webDAVFileInfo) ModTime() time.Time {
	// TODO fix it
	return time.Now()
}

func (wdfi *webDAVFileInfo) IsDir() bool {
	return wdfi.isDir
}

func (wdfi *webDAVFileInfo) Sys() interface{} {
	return nil
}
