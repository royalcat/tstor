package torrent

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/types"
)

type Service struct {
	c *torrent.Client

	// stats *Stats
	DefaultPriority types.PiecePriority

	log                     *slog.Logger
	addTimeout, readTimeout int
}

func NewService(c *torrent.Client, addTimeout, readTimeout int) *Service {
	l := slog.With("component", "torrent-service")
	return &Service{
		log:             l,
		c:               c,
		DefaultPriority: types.PiecePriorityNone,
		// stats:       newStats(), // TODO persistent
		addTimeout:  addTimeout,
		readTimeout: readTimeout,
	}
}

var _ vfs.FsFactory = (*Service)(nil).NewTorrentFs

func (s *Service) NewTorrentFs(f vfs.File) (vfs.Filesystem, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*time.Duration(s.addTimeout))
	defer cancel()
	defer f.Close()

	mi, err := metainfo.Load(f)
	if err != nil {
		return nil, err
	}

	t, ok := s.c.Torrent(mi.HashInfoBytes())
	if !ok {
		t, err = s.c.AddTorrent(mi)
		if err != nil {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("creating torrent fs timed out")
		case <-t.GotInfo():
		}
		for _, f := range t.Files() {
			f.SetPriority(s.DefaultPriority)
		}
		t.AllowDataDownload()
	}

	return vfs.NewTorrentFs(t, s.readTimeout), nil
}

func (s *Service) Stats() (*Stats, error) {
	return &Stats{}, nil
}
