package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"git.kmsign.ru/royalcat/tstor/src/config"
	"github.com/anacrolix/torrent/storage"
)

func SetupStorage(cfg config.TorrentClient) (*FileStorage, storage.PieceCompletion, error) {
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
	st := NewFileStorage(filesDir, pc)

	return st, pc, nil
}
