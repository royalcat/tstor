package host

import (
	"git.kmsign.ru/royalcat/tstor/src/host/service"
	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
)

func NewStorage(dataPath string, tsrv *service.Service) vfs.Filesystem {
	factories := map[string]vfs.FsFactory{
		".torrent": tsrv.NewTorrentFs,
	}

	// add default torrent factory for root filesystem
	for k, v := range vfs.ArchiveFactories {
		factories[k] = v
	}

	return vfs.NewResolveFS(vfs.NewOsFs(dataPath), factories)
}
