package torrent

import (
	"sync"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Service struct {
	c *torrent.Client

	// stats *Stats

	mu sync.Mutex

	log                     zerolog.Logger
	addTimeout, readTimeout int
}

func NewService(c *torrent.Client, addTimeout, readTimeout int) *Service {
	l := log.Logger.With().Str("component", "torrent-service").Logger()
	return &Service{
		log: l,
		c:   c,
		// stats:       newStats(), // TODO persistent
		addTimeout:  addTimeout,
		readTimeout: readTimeout,
	}
}

var _ vfs.FsFactory = (*Service)(nil).NewTorrentFs

func (s *Service) NewTorrentFs(f vfs.File) (vfs.Filesystem, error) {
	defer f.Close()

	mi, err := metainfo.Load(f)
	if err != nil {
		return nil, err
	}
	t, err := s.c.AddTorrent(mi)
	if err != nil {
		return nil, err
	}
	<-t.GotInfo()
	t.AllowDataDownload()
	for _, f := range t.Files() {
		f.SetPriority(torrent.PiecePriorityReadahead)
	}

	return vfs.NewTorrentFs(t, s.readTimeout), nil
}

func (s *Service) Stats() (*Stats, error) {
	return &Stats{}, nil
}

// func (s *Service) Load() (map[string]vfs.Filesystem, error) {
// 	// Load from config
// 	s.log.Info().Msg("adding torrents from configuration")
// 	for _, loader := range s.loaders {
// 		if err := s.load(loader); err != nil {
// 			return nil, err
// 		}
// 	}

// 	// Load from DB
// 	s.log.Info().Msg("adding torrents from database")
// 	return s.fss, s.load(s.db)
// }

// func (s *Service) load(l loader.Loader) error {
// 	list, err := l.ListMagnets()
// 	if err != nil {
// 		return err
// 	}
// 	for r, ms := range list {
// 		s.addRoute(r)
// 		for _, m := range ms {
// 			if err := s.addMagnet(r, m); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	list, err = l.ListTorrentPaths()
// 	if err != nil {
// 		return err
// 	}
// 	for r, ms := range list {
// 		s.addRoute(r)
// 		for _, p := range ms {
// 			if err := s.addTorrentPath(r, p); err != nil {
// 				return err
// 			}
// 		}
// 	}

// 	return nil
// }

// func (s *Service) AddMagnet(r, m string) error {
// 	if err := s.addMagnet(r, m); err != nil {
// 		return err
// 	}

// 	// Add to db
// 	return s.db.AddMagnet(r, m)
// }

// func (s *Service) addTorrentPath(r, p string) error {
// 	// Add to client
// 	t, err := s.c.AddTorrentFromFile(p)
// 	if err != nil {
// 		return err
// 	}

// 	return s.addTorrent(r, t)
// }

// func (s *Service) addMagnet(r, m string) error {
// 	// Add to client
// 	t, err := s.c.AddMagnet(m)
// 	if err != nil {
// 		return err
// 	}

// 	return s.addTorrent(r, t)

// }

// func (s *Service) addRoute(r string) {
// 	s.s.AddRoute(r)

// 	// Add to filesystems
// 	folder := path.Join("/", r)
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	_, ok := s.fss[folder]
// 	if !ok {
// 		s.fss[folder] = vfs.NewTorrentFs(s.readTimeout)
// 	}
// }

// func (s *Service) addTorrent(r string, t *torrent.Torrent) error {
// 	// only get info if name is not available
// 	if t.Info() == nil {
// 		s.log.Info().Str("hash", t.InfoHash().String()).Msg("getting torrent info")
// 		select {
// 		case <-time.After(time.Duration(s.addTimeout) * time.Second):
// 			s.log.Error().Str("hash", t.InfoHash().String()).Msg("timeout getting torrent info")
// 			return errors.New("timeout getting torrent info")
// 		case <-t.GotInfo():
// 			s.log.Info().Str("hash", t.InfoHash().String()).Msg("obtained torrent info")
// 		}

// 	}

// 	// Add to stats
// 	s.s.Add(r, t)

// 	// Add to filesystems
// 	folder := path.Join("/", r)
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	tfs, ok := s.fss[folder].(*vfs.TorrentFs)
// 	if !ok {
// 		return errors.New("error adding torrent to filesystem")
// 	}

// 	tfs.AddTorrent(t)
// 	s.log.Info().Str("name", t.Info().Name).Str("route", r).Msg("torrent added")

// 	return nil
// }

// func (s *Service) RemoveFromHash(r, h string) error {
// 	// Remove from db
// 	deleted, err := s.db.RemoveFromHash(r, h)
// 	if err != nil {
// 		return err
// 	}

// 	if !deleted {
// 		return fmt.Errorf("element with hash %v on route %v cannot be removed", h, r)
// 	}

// 	// Remove from stats
// 	s.s.Del(r, h)

// 	// Remove from fs
// 	folder := path.Join("/", r)

// 	tfs, ok := s.fss[folder].(*vfs.TorrentFs)
// 	if !ok {
// 		return errors.New("error removing torrent from filesystem")
// 	}

// 	tfs.RemoveTorrent(h)

// 	// Remove from client
// 	var mh metainfo.Hash
// 	if err := mh.FromHexString(h); err != nil {
// 		return err
// 	}

// 	t, ok := s.c.Torrent(metainfo.NewHashFromHex(h))
// 	if ok {
// 		t.Drop()
// 	}

// 	return nil
// }
