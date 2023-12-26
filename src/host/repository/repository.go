package repository

import (
	"errors"
	"path/filepath"
	"sync"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/badgerdb"
	"github.com/philippgille/gokv/encoding"
)

type TorrentsRepository interface {
	ExcludeFile(hash metainfo.Hash, file ...string) error
	ExcludedFiles(hash metainfo.Hash) ([]string, error)
}

func NewTorrentMetaRepository(dir string) (TorrentsRepository, error) {
	excludedFilesStore, err := badgerdb.NewStore(badgerdb.Options{
		Dir:   filepath.Join(dir, "excluded-files"),
		Codec: encoding.JSON,
	})

	if err != nil {
		return nil, err
	}

	r := &torrentRepositoryImpl{
		excludedFiles: excludedFilesStore,
	}

	return r, nil
}

type torrentRepositoryImpl struct {
	m             sync.RWMutex
	excludedFiles gokv.Store
}

var ErrNotFound = errors.New("not found")

func (r *torrentRepositoryImpl) ExcludeFile(hash metainfo.Hash, file ...string) error {
	r.m.Lock()
	defer r.m.Unlock()

	var excludedFiles []string
	found, err := r.excludedFiles.Get(hash.AsString(), &excludedFiles)
	if err != nil {
		return err
	}
	if !found {
		excludedFiles = []string{}
	}
	excludedFiles = unique(append(excludedFiles, file...))

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
