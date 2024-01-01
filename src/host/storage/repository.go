package storage

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	atstorage "github.com/anacrolix/torrent/storage"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/badgerdb"
	"github.com/philippgille/gokv/encoding"
)

type TorrentsRepository interface {
	ExcludeFile(file *torrent.File) error
	ExcludedFiles(hash metainfo.Hash) ([]string, error)
}

func NewTorrentMetaRepository(metaDir string, storage atstorage.ClientImplCloser) (TorrentsRepository, error) {
	excludedFilesStore, err := badgerdb.NewStore(badgerdb.Options{
		Dir:   filepath.Join(metaDir, "excluded-files"),
		Codec: encoding.JSON,
	})

	if err != nil {
		return nil, err
	}

	r := &torrentRepositoryImpl{
		excludedFiles: excludedFilesStore,
		storage:       storage,
	}

	return r, nil
}

type torrentRepositoryImpl struct {
	m             sync.RWMutex
	excludedFiles gokv.Store
	storage       atstorage.ClientImplCloser
}

var ErrNotFound = errors.New("not found")

func (r *torrentRepositoryImpl) ExcludeFile(file *torrent.File) error {
	r.m.Lock()
	defer r.m.Unlock()

	hash := file.Torrent().InfoHash()
	var excludedFiles []string
	found, err := r.excludedFiles.Get(hash.AsString(), &excludedFiles)
	if err != nil {
		return err
	}
	if !found {
		excludedFiles = []string{}
	}
	excludedFiles = unique(append(excludedFiles, file.Path()))

	if storage, ok := r.storage.(FileStorageDeleter); ok {
		err = storage.DeleteFile(file)
		if err != nil {
			return err
		}
	}

	return r.excludedFiles.Set(hash.AsString(), excludedFiles)
}

func (r *torrentRepositoryImpl) ExcludedFiles(hash metainfo.Hash) ([]string, error) {
	r.m.Lock()
	defer r.m.Unlock()

	var excludedFiles []string
	found, err := r.excludedFiles.Get(hash.AsString(), &excludedFiles)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	return excludedFiles, nil
}

func unique[C comparable](intSlice []C) []C {
	keys := make(map[C]bool)
	list := []C{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
