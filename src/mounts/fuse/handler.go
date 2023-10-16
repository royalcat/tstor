package fuse

import (
	"os"
	"path/filepath"
	"runtime"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/billziss-gh/cgofuse/fuse"

	"github.com/rs/zerolog/log"
)

type Handler struct {
	fuseAllowOther bool
	path           string

	host *fuse.FileSystemHost
}

func NewHandler(fuseAllowOther bool, path string) *Handler {
	return &Handler{
		fuseAllowOther: fuseAllowOther,
		path:           path,
	}
}

func (s *Handler) Mount(vfs vfs.Filesystem) error {
	folder := s.path
	// On windows, the folder must don't exist
	if runtime.GOOS == "windows" {
		folder = filepath.Dir(s.path)
	}

	if filepath.VolumeName(folder) == "" {
		if err := os.MkdirAll(folder, 0744); err != nil && !os.IsExist(err) {
			return err
		}
	}

	host := fuse.NewFileSystemHost(NewFS(vfs))

	// TODO improve error handling here
	go func() {
		var config []string

		if s.fuseAllowOther {
			config = append(config, "-o", "allow_other")
		}

		ok := host.Mount(s.path, config)
		if !ok {
			log.Error().Str("path", s.path).Msg("error trying to mount filesystem")
		}
	}()

	s.host = host

	log.Info().Str("path", s.path).Msg("starting FUSE mount")

	return nil
}

func (s *Handler) Unmount() {
	if s.host == nil {
		return
	}

	ok := s.host.Unmount()
	if !ok {
		//TODO try to force unmount if possible
		log.Error().Str("path", s.path).Msg("unmount failed")
	}
}
