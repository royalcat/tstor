package vfs

import (
	"context"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/host/storage"
	"git.kmsign.ru/royalcat/tstor/src/iio"
	"github.com/anacrolix/missinggo/v2"
	"github.com/anacrolix/torrent"
	"golang.org/x/exp/maps"
)

var _ Filesystem = &TorrentFs{}

type TorrentFs struct {
	mu  sync.Mutex
	t   *torrent.Torrent
	rep storage.TorrentsRepository

	readTimeout int

	//cache
	filesCache map[string]*torrentFile

	resolver *resolver
}

func NewTorrentFs(t *torrent.Torrent, rep storage.TorrentsRepository, readTimeout int) *TorrentFs {
	return &TorrentFs{
		t:           t,
		rep:         rep,
		readTimeout: readTimeout,
		resolver:    newResolver(ArchiveFactories),
	}
}

func (fs *TorrentFs) files() (map[string]*torrentFile, error) {
	if fs.filesCache == nil {
		fs.mu.Lock()
		<-fs.t.GotInfo()
		files := fs.t.Files()

		excludedFiles, err := fs.rep.ExcludedFiles(fs.t.InfoHash())
		if err != nil {
			return nil, err
		}

		fs.filesCache = make(map[string]*torrentFile)
		for _, file := range files {

			p := file.Path()

			if slices.Contains(excludedFiles, p) {
				continue
			}
			if strings.Contains(p, "/.pad/") {
				continue
			}

			p = AbsPath(file.Path())

			// TODO make optional
			// removing the torrent root directory of same name  as torrent
			p, _ = strings.CutPrefix(p, "/"+fs.t.Name()+"/")
			p = AbsPath(p)

			fs.filesCache[p] = &torrentFile{
				name:    path.Base(p),
				timeout: fs.readTimeout,
				file:    file,
			}
		}
		fs.mu.Unlock()
	}

	return fs.filesCache, nil
}

func (fs *TorrentFs) rawOpen(path string) (File, error) {
	files, err := fs.files()
	if err != nil {
		return nil, err
	}
	file, err := getFile(files, path)
	return file, err
}

func (fs *TorrentFs) rawStat(filename string) (fs.FileInfo, error) {
	files, err := fs.files()
	if err != nil {
		return nil, err
	}
	file, err := getFile(files, filename)
	if err != nil {
		return nil, err
	}
	if file.IsDir() {
		return newDirInfo(path.Base(filename)), nil
	} else {
		return newFileInfo(path.Base(filename), file.Size()), nil
	}

}

// Stat implements Filesystem.
func (fs *TorrentFs) Stat(filename string) (fs.FileInfo, error) {
	if filename == Separator {
		return newDirInfo(filename), nil
	}

	fsPath, nestedFs, nestedFsPath, err := fs.resolver.resolvePath(filename, fs.rawOpen)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.Stat(nestedFsPath)
	}

	return fs.rawStat(fsPath)
}

func (fs *TorrentFs) Open(filename string) (File, error) {
	fsPath, nestedFs, nestedFsPath, err := fs.resolver.resolvePath(filename, fs.rawOpen)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.Open(nestedFsPath)
	}

	return fs.rawOpen(fsPath)
}

func (fs *TorrentFs) ReadDir(name string) ([]fs.DirEntry, error) {
	fsPath, nestedFs, nestedFsPath, err := fs.resolver.resolvePath(name, fs.rawOpen)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.ReadDir(nestedFsPath)
	}
	files, err := fs.files()
	if err != nil {
		return nil, err
	}

	return listDirFromFiles(files, fsPath)
}

func (fs *TorrentFs) Unlink(name string) error {
	name = AbsPath(name)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	files, err := fs.files()
	if err != nil {
		return err
	}

	if !slices.Contains(maps.Keys(files), name) {
		return ErrNotExist
	}

	file := files[name]
	delete(fs.filesCache, name)

	return fs.rep.ExcludeFile(file.file)
}

type reader interface {
	iio.Reader
	missinggo.ReadContexter
}

type readAtWrapper struct {
	timeout int
	mu      sync.Mutex

	torrent.Reader
	io.ReaderAt
	io.Closer
}

func newReadAtWrapper(r torrent.Reader, timeout int) reader {
	w := &readAtWrapper{Reader: r, timeout: timeout}
	w.SetResponsive()
	return w
}

func (rw *readAtWrapper) ReadAt(p []byte, off int64) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	_, err := rw.Seek(off, io.SeekStart)
	if err != nil {
		return 0, err
	}

	return readAtLeast(rw, rw.timeout, p, len(p))
}

func readAtLeast(r missinggo.ReadContexter, timeout int, buf []byte, min int) (n int, err error) {
	if len(buf) < min {
		return 0, io.ErrShortBuffer
	}
	for n < min && err == nil {
		var nn int

		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		nn, err = r.ReadContext(ctx, buf[n:])
		n += nn
	}
	if n >= min {
		err = nil
	} else if n > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

func (rw *readAtWrapper) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	return rw.Reader.Close()
}

var _ File = &torrentFile{}

type torrentFile struct {
	name string

	reader  reader
	timeout int

	file *torrent.File
}

func (d *torrentFile) Stat() (fs.FileInfo, error) {
	return newFileInfo(d.name, d.file.Length()), nil
}

func (d *torrentFile) load() {
	if d.reader != nil {
		return
	}
	d.reader = newReadAtWrapper(d.file.NewReader(), d.timeout)
}

func (d *torrentFile) Size() int64 {
	return d.file.Length()
}

func (d *torrentFile) IsDir() bool {
	return false
}

func (d *torrentFile) Close() error {
	var err error
	if d.reader != nil {
		err = d.reader.Close()
	}

	d.reader = nil

	return err
}

func (d *torrentFile) Read(p []byte) (n int, err error) {
	d.load()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(d.timeout)*time.Second)
	defer cancel()

	return d.reader.ReadContext(ctx, p)
}

func (d *torrentFile) ReadAt(p []byte, off int64) (n int, err error) {
	d.load()
	return d.reader.ReadAt(p, off)
}
