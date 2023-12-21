package torrent

import (
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	tlog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/rs/zerolog/log"

	"git.kmsign.ru/royalcat/tstor/src/config"
	dlog "git.kmsign.ru/royalcat/tstor/src/log"
)

func NewClient(st storage.ClientImpl, fis bep44.Store, cfg *config.TorrentClient, id [20]byte) (*torrent.Client, error) {
	// TODO download and upload limits
	torrentCfg := torrent.NewDefaultClientConfig()
	torrentCfg.PeerID = string(id[:])
	torrentCfg.DefaultStorage = st

	// torrentCfg.DisableIPv6 = cfg.DisableIPv6
	// torrentCfg.DropDuplicatePeerIds = true
	// torrentCfg.TorrentPeersLowWater = 10
	// torrentCfg.TorrentPeersHighWater = 100
	// torrentCfg.DisableWebtorrent = true
	// torrentCfg.DisableAggressiveUpload = true
	// torrentCfg.DisableWebseeds = true
	// torrentCfg.DisableUTP = false
	// torrentCfg.NoDefaultPortForwarding = true
	// torrentCfg.AlwaysWantConns = false
	// torrentCfg.ClientDhtConfig = torrent.ClientDhtConfig{
	// 	NoDHT: true,
	// }

	l := log.Logger.With().Str("component", "torrent-client").Logger()

	tl := tlog.NewLogger()
	tl.SetHandlers(&dlog.Torrent{L: l})
	torrentCfg.Logger = tl

	torrentCfg.ConfigureAnacrolixDhtServer = func(cfg *dht.ServerConfig) {
		cfg.Store = fis
		cfg.Exp = 2 * time.Hour
		cfg.NoSecurity = false
	}

	return torrent.NewClient(torrentCfg)
}
