package storage

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/anacrolix/missinggo"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/common"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/segments"
	"github.com/anacrolix/torrent/storage"
)

type FileStorageDeleter interface {
	storage.ClientImplCloser
	DeleteFile(file *torrent.File) error
}

// NewFileStorage creates a new ClientImplCloser that stores files using the OS native filesystem.
func NewFileStorage(baseDir string, pc storage.PieceCompletion) FileStorageDeleter {
	return &FileStorage{baseDir: baseDir, pieceCompletion: pc}
}

// File-based storage for torrents, that isn't yet bound to a particular torrent.
type FileStorage struct {
	baseDir         string
	pieceCompletion storage.PieceCompletion
}

func (me *FileStorage) Close() error {
	return me.pieceCompletion.Close()
}

func (me *FileStorage) torrentDir(info *metainfo.Info, infoHash metainfo.Hash) string {
	return filepath.Join(me.baseDir, info.Name)
}

func (me *FileStorage) filePath(file metainfo.FileInfo) string {
	return filepath.Join(file.Path...)
}

func (fs *FileStorage) DeleteFile(file *torrent.File) error {
	info := file.Torrent().Info()
	infoHash := file.Torrent().InfoHash()
	torrentDir := fs.torrentDir(info, infoHash)
	relFilePath := fs.filePath(file.FileInfo())
	filePath := path.Join(torrentDir, relFilePath)
	for i := file.BeginPieceIndex(); i < file.EndPieceIndex(); i++ {
		pk := metainfo.PieceKey{InfoHash: infoHash, Index: i}
		err := fs.pieceCompletion.Set(pk, false)
		if err != nil {
			return err
		}
	}
	return os.Remove(filePath)
}

func (fs FileStorage) OpenTorrent(info *metainfo.Info, infoHash metainfo.Hash) (storage.TorrentImpl, error) {
	dir := fs.torrentDir(info, infoHash)
	upvertedFiles := info.UpvertedFiles()
	files := make([]file, 0, len(upvertedFiles))
	for i, fileInfo := range upvertedFiles {
		filePath := filepath.Join(dir, fs.filePath(fileInfo))
		if !isSubFilepath(dir, filePath) {
			return storage.TorrentImpl{}, fmt.Errorf("file %v: path %q is not sub path of %q", i, filePath, fs.baseDir)
		}

		f := file{
			path:   filePath,
			length: fileInfo.Length,
		}
		if f.length == 0 {
			err := CreateNativeZeroLengthFile(f.path)
			if err != nil {
				return storage.TorrentImpl{}, fmt.Errorf("creating zero length file: %w", err)
			}
		}
		files = append(files, f)
	}
	t := &fileTorrentImpl{
		files:          files,
		segmentLocater: segments.NewIndex(common.LengthIterFromUpvertedFiles(upvertedFiles)),
		infoHash:       infoHash,
		completion:     fs.pieceCompletion,
	}
	return storage.TorrentImpl{
		Piece: t.Piece,
		Close: t.Close,
	}, nil
}

type file struct {
	// The safe, OS-local file path.
	path   string
	length int64
}

type fileTorrentImpl struct {
	files          []file
	segmentLocater segments.Index
	infoHash       metainfo.Hash
	completion     storage.PieceCompletion
}

func (fts *fileTorrentImpl) Piece(p metainfo.Piece) storage.PieceImpl {
	// Create a view onto the file-based torrent storage.
	_io := fileTorrentImplIO{fts}
	// Return the appropriate segments of this.
	return &filePieceImpl{
		fileTorrentImpl: fts,
		p:               p,
		WriterAt:        missinggo.NewSectionWriter(_io, p.Offset(), p.Length()),
		ReaderAt:        io.NewSectionReader(_io, p.Offset(), p.Length()),
	}
}

func (fs *fileTorrentImpl) Close() error {
	return nil
}

// A helper to create zero-length files which won't appear for file-orientated storage since no
// writes will ever occur to them (no torrent data is associated with a zero-length file). The
// caller should make sure the file name provided is safe/sanitized.
func CreateNativeZeroLengthFile(name string) error {
	err := os.MkdirAll(filepath.Dir(name), 0o777)
	if err != nil {
		return err
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	return f.Close()
}

// Exposes file-based storage of a torrent, as one big ReadWriterAt.
type fileTorrentImplIO struct {
	fts *fileTorrentImpl
}

// Returns EOF on short or missing file.
func (fst *fileTorrentImplIO) readFileAt(file file, b []byte, off int64) (n int, err error) {
	f, err := os.Open(file.path)
	if os.IsNotExist(err) {
		// File missing is treated the same as a short file.
		err = io.EOF
		return
	}
	if err != nil {
		return
	}
	defer f.Close()
	// Limit the read to within the expected bounds of this file.
	if int64(len(b)) > file.length-off {
		b = b[:file.length-off]
	}
	for off < file.length && len(b) != 0 {
		n1, err1 := f.ReadAt(b, off)
		b = b[n1:]
		n += n1
		off += int64(n1)
		if n1 == 0 {
			err = err1
			break
		}
	}
	return
}

// Only returns EOF at the end of the torrent. Premature EOF is ErrUnexpectedEOF.
func (fst fileTorrentImplIO) ReadAt(b []byte, off int64) (n int, err error) {
	fst.fts.segmentLocater.Locate(
		segments.Extent{Start: off, Length: int64(len(b))},
		func(i int, e segments.Extent) bool {
			n1, err1 := fst.readFileAt(fst.fts.files[i], b[:e.Length], e.Start)
			n += n1
			b = b[n1:]
			err = err1
			return err == nil // && int64(n1) == e.Length
		},
	)
	if len(b) != 0 && err == nil {
		err = io.EOF
	}
	return
}

func (fst fileTorrentImplIO) WriteAt(p []byte, off int64) (n int, err error) {
	// log.Printf("write at %v: %v bytes", off, len(p))
	fst.fts.segmentLocater.Locate(
		segments.Extent{Start: off, Length: int64(len(p))},
		func(i int, e segments.Extent) bool {
			name := fst.fts.files[i].path
			err = os.MkdirAll(filepath.Dir(name), 0o777)
			if err != nil {
				return false
			}
			var f *os.File
			f, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0o666)
			if err != nil {
				return false
			}
			var n1 int
			n1, err = f.WriteAt(p[:e.Length], e.Start)
			// log.Printf("%v %v wrote %v: %v", i, e, n1, err)
			closeErr := f.Close()
			n += n1
			p = p[n1:]
			if err == nil {
				err = closeErr
			}
			if err == nil && int64(n1) != e.Length {
				err = io.ErrShortWrite
			}
			return err == nil
		},
	)
	return n, err
}

type filePieceImpl struct {
	*fileTorrentImpl
	p metainfo.Piece
	io.WriterAt
	io.ReaderAt
}

var _ storage.PieceImpl = (*filePieceImpl)(nil)

func (me *filePieceImpl) pieceKey() metainfo.PieceKey {
	return metainfo.PieceKey{InfoHash: me.infoHash, Index: me.p.Index()}
}

func (fs *filePieceImpl) Completion() storage.Completion {
	c, err := fs.completion.Get(fs.pieceKey())
	if err != nil {
		log.Printf("error getting piece completion: %s", err)
		c.Ok = false
		return c
	}

	verified := true
	if c.Complete {
		// If it's allegedly complete, check that its constituent files have the necessary length.
		for _, fi := range extentCompleteRequiredLengths(fs.p.Info, fs.p.Offset(), fs.p.Length()) {
			s, err := os.Stat(fs.files[fi.fileIndex].path)
			if err != nil || s.Size() < fi.length {
				verified = false
				break
			}
		}
	}

	if !verified {
		// The completion was wrong, fix it.
		c.Complete = false
		fs.completion.Set(fs.pieceKey(), false)
	}

	return c
}

func (fs *filePieceImpl) MarkComplete() error {
	return fs.completion.Set(fs.pieceKey(), true)
}

func (fs *filePieceImpl) MarkNotComplete() error {
	return fs.completion.Set(fs.pieceKey(), false)
}

type requiredLength struct {
	fileIndex int
	length    int64
}

func isSubFilepath(base, sub string) bool {
	rel, err := filepath.Rel(base, sub)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func extentCompleteRequiredLengths(info *metainfo.Info, off, n int64) (ret []requiredLength) {
	if n == 0 {
		return
	}
	for i, fi := range info.UpvertedFiles() {
		if off >= fi.Length {
			off -= fi.Length
			continue
		}
		n1 := n
		if off+n1 > fi.Length {
			n1 = fi.Length - off
		}
		ret = append(ret, requiredLength{
			fileIndex: i,
			length:    off + n1,
		})
		n -= n1
		if n == 0 {
			return
		}
		off = 0
	}
	panic("extent exceeds torrent bounds")
}
