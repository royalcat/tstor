package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/config"
	"git.kmsign.ru/royalcat/tstor/src/host"
	"git.kmsign.ru/royalcat/tstor/src/host/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"git.kmsign.ru/royalcat/tstor/src/http"
	dlog "git.kmsign.ru/royalcat/tstor/src/log"
	"git.kmsign.ru/royalcat/tstor/src/mounts/fuse"
	"git.kmsign.ru/royalcat/tstor/src/mounts/httpfs"
	"git.kmsign.ru/royalcat/tstor/src/mounts/webdav"
)

const (
	configFlag     = "config"
	portFlag       = "http-port"
	webDAVPortFlag = "webdav-port"
)

func main() {
	app := &cli.App{
		Name:  "tstor",
		Usage: "Torrent client with on-demand file downloading as a filesystem.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  configFlag,
				Value: "./config.yaml",
				Usage: "YAML file containing tstor configuration.",
			},
		},

		Action: func(c *cli.Context) error {
			err := load(c.String(configFlag))

			// stop program execution on errors to avoid flashing consoles
			if err != nil && runtime.GOOS == "windows" {
				log.Error().Err(err).Msg("problem starting application")
				fmt.Print("Press 'Enter' to continue...")
				bufio.NewReader(os.Stdin).ReadBytes('\n')
			}

			return err
		},

		HideHelpCommand: true,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("problem starting application")
	}
}

func setupStorage(tcfg config.TorrentClient) (storage.ClientImplCloser, storage.PieceCompletion, error) {
	pcp := filepath.Join(tcfg.DataFolder, "piece-completion")
	if err := os.MkdirAll(pcp, 0744); err != nil {
		return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	}
	pc, err := storage.NewBoltPieceCompletion(pcp)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating servers piece completion: %w", err)
	}

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

	filesDir := filepath.Join(tcfg.DataFolder, "files")
	if err := os.MkdirAll(pcp, 0744); err != nil {
		return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	}

	st := storage.NewFileWithCompletion(filesDir, pc)

	return st, pc, nil
}

type stc struct {
	storage.ClientImpl
}

func (s *stc) Close() error {
	return nil
}

func load(configPath string) error {
	conf, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	dlog.Load(&conf.Log)

	if err := os.MkdirAll(conf.TorrentClient.MetadataFolder, 0744); err != nil {
		return fmt.Errorf("error creating metadata folder: %w", err)
	}

	fis, err := torrent.NewFileItemStore(filepath.Join(conf.TorrentClient.MetadataFolder, "items"), 2*time.Hour)
	if err != nil {
		return fmt.Errorf("error starting item store: %w", err)
	}

	id, err := torrent.GetOrCreatePeerID(filepath.Join(conf.TorrentClient.MetadataFolder, "ID"))
	if err != nil {
		return fmt.Errorf("error creating node ID: %w", err)
	}

	st, _, err := setupStorage(conf.TorrentClient)
	if err != nil {
		return err
	}

	c, err := torrent.NewClient(st, fis, &conf.TorrentClient, id)
	if err != nil {
		return fmt.Errorf("error starting torrent client: %w", err)
	}
	c.AddDhtNodes(conf.TorrentClient.DHTNodes)

	ts := torrent.NewService(c, conf.TorrentClient.AddTimeout, conf.TorrentClient.ReadTimeout)

	if err := os.MkdirAll(conf.DataFolder, 0744); err != nil {
		return fmt.Errorf("error creating data folder: %w", err)
	}
	cfs := host.NewStorage(conf.DataFolder, ts)

	var mh *fuse.Handler
	if conf.Mounts.Fuse.Enabled {
		mh = fuse.NewHandler(conf.Mounts.Fuse.AllowOther, conf.Mounts.Fuse.Path)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {

		<-sigChan
		log.Info().Msg("closing servers...")
		// for _, s := range servers {
		// 	if err := s.Close(); err != nil {
		// 		log.Warn().Err(err).Msg("problem closing server")
		// 	}
		// }
		log.Info().Msg("closing items database...")
		fis.Close()
		log.Info().Msg("closing torrent client...")
		c.Close()
		if mh != nil {
			log.Info().Msg("unmounting fuse filesystem...")
			mh.Unmount()
		}

		log.Info().Msg("exiting")
		os.Exit(1)
	}()

	go func() {
		if mh == nil {
			return
		}

		if err := mh.Mount(cfs); err != nil {
			log.Info().Err(err).Msg("error mounting filesystems")
		}
	}()

	if conf.Mounts.WebDAV.Enabled {
		go func() {
			if err := webdav.NewWebDAVServer(cfs, conf.Mounts.WebDAV.Port, conf.Mounts.WebDAV.User, conf.Mounts.WebDAV.Pass); err != nil {
				log.Error().Err(err).Msg("error starting webDAV")
			}

			log.Warn().Msg("webDAV configuration not found!")
		}()
	}
	if conf.Mounts.HttpFs.Enabled {
		go func() {
			httpfs := httpfs.NewHTTPFS(cfs)

			r := gin.New()

			r.GET("*filepath", func(c *gin.Context) {
				path := c.Param("filepath")
				c.FileFromFS(path, httpfs)
			})

			log.Info().Str("host", fmt.Sprintf("0.0.0.0:%d", conf.Mounts.HttpFs.Port)).Msg("starting HTTPFS")
			if err := r.Run(fmt.Sprintf("0.0.0.0:%d", conf.Mounts.HttpFs.Port)); err != nil {
				log.Error().Err(err).Msg("error starting HTTPFS")
			}
		}()
	}

	logFilename := filepath.Join(conf.Log.Path, dlog.FileName)

	err = http.New(nil, nil, ts, logFilename, conf)
	log.Error().Err(err).Msg("error initializing HTTP server")
	return err
}
