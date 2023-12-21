package torrent

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"git.kmsign.ru/royalcat/tstor/src/config"
	"github.com/anacrolix/missinggo"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/mmap_span"
	"github.com/anacrolix/torrent/storage"
	"github.com/edsrzf/mmap-go"
)

type Torrent struct {
	client *torrent.Client
	data   storage.ClientImplCloser
	pc     storage.PieceCompletion
}

func SetupStorage(cfg config.TorrentClient) (storage.ClientImplCloser, storage.PieceCompletion, error) {
	pcp := filepath.Join(cfg.DataFolder, "piece-completion")
	if err := os.MkdirAll(pcp, 0744); err != nil {
		return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	}
	pc, err := storage.NewBoltPieceCompletion(pcp)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating servers piece completion: %w", err)
	}

	// pc, err := NewBadgerPieceCompletion(pcp)
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("error creating servers piece completion: %w", err)
	// }

	// TODO implement cache/storage switching
	// cacheDir := filepath.Join(tcfg.DataFolder, "cache")
	// if err := os.MkdirAll(cacheDir, 0744); err != nil {
	// 	return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	// }
	// fc, err := filecache.NewCache(cacheDir)
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("error creating cache: %w", err)
	// }
	// log.Info().Msg(fmt.Sprintf("setting cache size to %d MB", 1024))
	// fc.SetCapacity(1024 * 1024 * 1024)

	// rp := storage.NewResourcePieces(fc.AsResourceProvider())
	// st := &stc{rp}

	filesDir := filepath.Join(cfg.DataFolder, "files")
	if err := os.MkdirAll(pcp, 0744); err != nil {
		return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	}

	// st := storage.NewMMapWithCompletion(filesDir, pc)
	st := storage.NewFileOpts(storage.NewFileClientOpts{
		ClientBaseDir:   filesDir,
		PieceCompletion: pc,
	})

	return st, pc, nil
}

func (s Torrent) Remove(f *torrent.File) error {

	return nil
}

// type dupePieces struct {
// }

// func (s Torrent) dedupe(f1, f2 *os.File) error {
// 	for _, t := range s.client.Torrents() {
// 		for i := 0; i < t.NumPieces(); i++ {
// 			p := t.Piece(i)
// 			p.Info().Hash()
// 		}
// 	}

// 	// https://go-review.googlesource.com/c/sys/+/284352/10/unix/syscall_linux_test.go#856
// 	// dedupe := unix.FileDedupeRange{
// 	// 	Src_offset: uint64(0),
// 	// 	Src_length: uint64(4096),
// 	// 	Info: []unix.FileDedupeRangeInfo{
// 	// 		unix.FileDedupeRangeInfo{
// 	// 			Dest_fd:     int64(f2.Fd()),
// 	// 			Dest_offset: uint64(0),
// 	// 		},
// 	// 		unix.FileDedupeRangeInfo{
// 	// 			Dest_fd:     int64(f2.Fd()),
// 	// 			Dest_offset: uint64(4096),
// 	// 		},
// 	// 	}}
// 	// err := unix.IoctlFileDedupeRange(int(f1.Fd()), &dedupe)
// 	// if err == unix.EOPNOTSUPP || err == unix.EINVAL {
// 	// 	t.Skip("deduplication not supported on this filesystem")
// 	// } else if err != nil {
// 	// 	t.Fatal(err)
// 	// }

// 	return nil
// }

type mmapClientImpl struct {
	baseDir string
	pc      storage.PieceCompletion
}

func NewMMapWithCompletion(baseDir string, completion storage.PieceCompletion) *mmapClientImpl {
	return &mmapClientImpl{
		baseDir: baseDir,
		pc:      completion,
	}
}

func (s *mmapClientImpl) OpenTorrent(info *metainfo.Info, infoHash metainfo.Hash) (_ storage.TorrentImpl, err error) {
	t, err := newMMapTorrent(info, infoHash, s.baseDir, s.pc)
	if err != nil {
		return storage.TorrentImpl{}, err
	}
	return storage.TorrentImpl{Piece: t.Piece, Close: t.Close, Flush: t.Flush}, nil
}

func (s *mmapClientImpl) Close() error {
	return s.pc.Close()
}

func newMMapTorrent(md *metainfo.Info, infoHash metainfo.Hash, location string, pc storage.PieceCompletionGetSetter) (*mmapTorrent, error) {
	span := &mmap_span.MMapSpan{}
	basePath, err := storage.ToSafeFilePath(md.Name)
	if err != nil {
		return nil, err
	}
	basePath = filepath.Join(location, basePath)

	for _, miFile := range md.UpvertedFiles() {
		var safeName string
		safeName, err = storage.ToSafeFilePath(miFile.Path...)
		if err != nil {
			return nil, err
		}
		fileName := filepath.Join(basePath, safeName)
		var mm FileMapping
		mm, err = mmapFile(fileName, miFile.Length)
		if err != nil {
			err = fmt.Errorf("file %q: %s", miFile.DisplayPath(md), err)
			return nil, err
		}
		span.Append(mm)
	}
	span.InitIndex()

	return &mmapTorrent{
		infoHash: infoHash,
		span:     span,
		pc:       pc,
	}, nil
}

type mmapTorrent struct {
	infoHash metainfo.Hash
	span     *mmap_span.MMapSpan
	pc       storage.PieceCompletionGetSetter
}

func (ts *mmapTorrent) Piece(p metainfo.Piece) storage.PieceImpl {
	return mmapPiece{
		pc:       ts.pc,
		p:        p,
		ih:       ts.infoHash,
		ReaderAt: io.NewSectionReader(ts.span, p.Offset(), p.Length()),
		WriterAt: missinggo.NewSectionWriter(ts.span, p.Offset(), p.Length()),
	}
}

func (ts *mmapTorrent) Close() error {
	errs := ts.span.Close()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (ts *mmapTorrent) Flush() error {
	errs := ts.span.Flush()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

type mmapPiece struct {
	pc storage.PieceCompletionGetSetter
	p  metainfo.Piece
	ih metainfo.Hash
	io.ReaderAt
	io.WriterAt
}

func (me mmapPiece) pieceKey() metainfo.PieceKey {
	return metainfo.PieceKey{InfoHash: me.ih, Index: me.p.Index()}
}

func (sp mmapPiece) Completion() storage.Completion {
	c, err := sp.pc.Get(sp.pieceKey())
	if err != nil {
		panic(err)
	}
	return c
}

func (sp mmapPiece) MarkComplete() error {
	return sp.pc.Set(sp.pieceKey(), true)
}

func (sp mmapPiece) MarkNotComplete() error {
	return sp.pc.Set(sp.pieceKey(), false)
}

func mmapFile(name string, size int64) (_ FileMapping, err error) {
	dir := filepath.Dir(name)
	err = os.MkdirAll(dir, 0o750)
	if err != nil {
		return nil, fmt.Errorf("making directory %q: %s", dir, err)
	}
	var file *os.File
	file, err = os.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			file.Close()
		}
	}()
	var fi os.FileInfo
	fi, err = file.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() < size {
		// I think this is necessary on HFS+. Maybe Linux will SIGBUS too if
		// you overmap a file but I'm not sure.
		err = file.Truncate(size)
		if err != nil {
			return nil, err
		}
	}
	return func() (ret mmapWithFile, err error) {
		ret.f = file
		if size == 0 {
			// Can't mmap() regions with length 0.
			return
		}
		intLen := int(size)
		if int64(intLen) != size {
			err = errors.New("size too large for system")
			return
		}
		ret.mmap, err = mmap.MapRegion(file, intLen, mmap.RDWR, 0, 0)
		if err != nil {
			err = fmt.Errorf("error mapping region: %s", err)
			return
		}
		if int64(len(ret.mmap)) != size {
			panic(len(ret.mmap))
		}
		return
	}()
}

type FileMapping = mmap_span.Mmap

// Handles closing the mmap's file handle (needed for Windows). Could be implemented differently by
// OS.
type mmapWithFile struct {
	f    *os.File
	mmap mmap.MMap
}

func (m mmapWithFile) Flush() error {
	return m.mmap.Flush()
}

func (m mmapWithFile) Unmap() (err error) {
	if m.mmap != nil {
		err = m.mmap.Unmap()
	}
	fileErr := m.f.Close()
	if err == nil {
		err = fileErr
	}
	return
}

func (m mmapWithFile) Bytes() []byte {
	if m.mmap == nil {
		return nil
	}
	return m.mmap
}
