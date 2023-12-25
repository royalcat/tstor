package repository

import (
	"errors"
	"sync"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/philippgille/gokv"
	"github.com/philippgille/gokv/badgerdb"
	"github.com/philippgille/gokv/encoding"
)

type TorrentMetaRepository interface {
	ExcludeFile(hash metainfo.Hash, file ...string) error
	ExcludedFiles(hash metainfo.Hash) ([]string, error)
}

func NewTorrentMetaRepository(dir string) (TorrentMetaRepository, error) {
	store, err := badgerdb.NewStore(badgerdb.Options{
		Dir:   dir,
		Codec: encoding.JSON,
	})
	if err != nil {
		return nil, err
	}

	r := &torrentRepositoryImpl{
		store: store,
	}

	return r, nil
}

type torrentRepositoryImpl struct {
	m     sync.RWMutex
	store gokv.Store
}

type torrentMeta struct {
	ExludedFiles []string
}

var ErrNotFound = errors.New("not found")

func (r *torrentRepositoryImpl) ExcludeFile(hash metainfo.Hash, file ...string) error {
	r.m.Lock()
	defer r.m.Unlock()

	var meta torrentMeta
	found, err := r.store.Get(hash.AsString(), &meta)
	if err != nil {
		return err
	}
	if !found {
		meta = torrentMeta{
			ExludedFiles: file,
		}
	}
	meta.ExludedFiles = unique(append(meta.ExludedFiles, file...))

	return r.store.Set(hash.AsString(), meta)
}

func (r *torrentRepositoryImpl) ExcludedFiles(hash metainfo.Hash) ([]string, error) {
	r.m.Lock()
	defer r.m.Unlock()

	var meta torrentMeta
	found, err := r.store.Get(hash.AsString(), &meta)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}

	return meta.ExludedFiles, nil
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
