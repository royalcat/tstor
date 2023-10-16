package vfs

import (
	"context"
	"io"
	"sync"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/iio"
	"github.com/anacrolix/missinggo/v2"
	"github.com/anacrolix/torrent"
)

var _ Filesystem = &TorrentFs{}

type TorrentFs struct {
	mu          sync.RWMutex
	t           *torrent.Torrent
	readTimeout int

	resolver *resolver
}

func NewTorrentFs(t *torrent.Torrent, readTimeout int) *TorrentFs {
	return &TorrentFs{
		t:           t,
		readTimeout: readTimeout,
		resolver:    newResolver(ArchiveFactories),
	}
}

func (fs *TorrentFs) files() map[string]*torrentFile {
	files := make(map[string]*torrentFile)
	<-fs.t.GotInfo()
	for _, file := range fs.t.Files() {
		p := clean(file.Path())
		files[p] = &torrentFile{
			readerFunc: file.NewReader,
			len:        file.Length(),
			timeout:    fs.readTimeout,
		}
	}

	return files
}

func (fs *TorrentFs) rawOpen(path string) (File, error) {
	file, err := getFile(fs.files(), path)
	return file, err
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

func (fs *TorrentFs) ReadDir(name string) (map[string]File, error) {
	fsPath, nestedFs, nestedFsPath, err := fs.resolver.resolvePath(name, fs.rawOpen)
	if err != nil {
		return nil, err
	}
	if nestedFs != nil {
		return nestedFs.ReadDir(nestedFsPath)
	}

	return listFilesInDir(fs.files(), fsPath)
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

		ctx, cancel := context.WithCancel(context.Background())
		timer := time.AfterFunc(
			time.Duration(timeout)*time.Second,
			func() {
				cancel()
			},
		)

		nn, err = r.ReadContext(ctx, buf[n:])
		n += nn

		timer.Stop()
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
	readerFunc func() torrent.Reader
	reader     reader
	len        int64
	timeout    int
}

func (d *torrentFile) load() {
	if d.reader != nil {
		return
	}
	d.reader = newReadAtWrapper(d.readerFunc(), d.timeout)
}

func (d *torrentFile) Size() int64 {
	return d.len
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
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(
		time.Duration(d.timeout)*time.Second,
		func() {
			cancel()
		},
	)

	defer timer.Stop()

	return d.reader.ReadContext(ctx, p)
}

func (d *torrentFile) ReadAt(p []byte, off int64) (n int, err error) {
	d.load()
	return d.reader.ReadAt(p, off)
}
