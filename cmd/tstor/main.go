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
	"github.com/anacrolix/torrent/storage"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"git.kmsign.ru/royalcat/tstor/src/fs"
	"git.kmsign.ru/royalcat/tstor/src/http"
	dlog "git.kmsign.ru/royalcat/tstor/src/log"
	"git.kmsign.ru/royalcat/tstor/src/mounts/fuse"
	"git.kmsign.ru/royalcat/tstor/src/mounts/httpfs"
	"git.kmsign.ru/royalcat/tstor/src/mounts/webdav"
	"git.kmsign.ru/royalcat/tstor/src/torrent"
	"git.kmsign.ru/royalcat/tstor/src/torrent/loader"
)

const (
	configFlag     = "config"
	fuseAllowOther = "fuse-allow-other"
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
			err := load(c.String(configFlag), c.Int(portFlag), c.Int(webDAVPortFlag), c.Bool(fuseAllowOther))

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
	pcp := filepath.Join(tcfg.MetadataFolder, "piece-completion")
	if err := os.MkdirAll(pcp, 0744); err != nil {
		return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	}
	pc, err := storage.NewBoltPieceCompletion(pcp)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating servers piece completion: %w", err)
	}

	// TODO implement cache dir and storage capacity
	// cacheDir := filepath.Join(tcfg.MetadataFolder, "cache")
	// if err := os.MkdirAll(cacheDir, 0744); err != nil {
	// 	return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	// }
	// fc, err := filecache.NewCache(cacheDir)
	// if err != nil {
	// 	return nil, nil, fmt.Errorf("error creating cache: %w", err)
	// }
	// log.Info().Msg(fmt.Sprintf("setting cache size to %d MB", tcfg.GlobalCacheSize))
	// fc.SetCapacity(tcfg.GlobalCacheSize * 1024 * 1024)

	filesDir := filepath.Join(tcfg.MetadataFolder, "files")
	if err := os.MkdirAll(pcp, 0744); err != nil {
		return nil, nil, fmt.Errorf("error creating piece completion folder: %w", err)
	}

	st := storage.NewFileWithCompletion(filesDir, pc)

	return st, pc, nil
}

func load(configPath string, port, webDAVPort int, fuseAllowOther bool) error {
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

	st, pc, err := setupStorage(conf.TorrentClient)
	if err != nil {
		return err
	}

	c, err := torrent.NewClient(st, fis, &conf.TorrentClient, id)
	if err != nil {
		return fmt.Errorf("error starting torrent client: %w", err)
	}

	var servers []*torrent.Server
	for _, s := range conf.TorrentClient.Servers {
		server := torrent.NewServer(c, pc, &s)
		servers = append(servers, server)
		if err := server.Start(); err != nil {
			return fmt.Errorf("error starting server: %w", err)
		}
	}

	cl := loader.NewConfig(conf.TorrentClient.Routes)
	fl := loader.NewFolder(conf.TorrentClient.Routes)
	ss := torrent.NewStats()

	dbl, err := loader.NewDB(filepath.Join(conf.TorrentClient.MetadataFolder, "magnetdb"))
	if err != nil {
		return fmt.Errorf("error starting magnet database: %w", err)
	}

	ts := torrent.NewService([]loader.Loader{cl, fl}, dbl, ss, c, conf.TorrentClient.AddTimeout, conf.TorrentClient.ReadTimeout)

	var mh *fuse.Handler
	if conf.Mounts.Fuse.Enabled {
		mh = fuse.NewHandler(conf.Mounts.Fuse.AllowOther, conf.Mounts.Fuse.Path)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {

		<-sigChan
		log.Info().Msg("closing servers...")
		for _, s := range servers {
			if err := s.Close(); err != nil {
				log.Warn().Err(err).Msg("problem closing server")
			}
		}
		log.Info().Msg("closing items database...")
		fis.Close()
		log.Info().Msg("closing magnet database...")
		dbl.Close()
		log.Info().Msg("closing torrent client...")
		c.Close()
		if mh != nil {
			log.Info().Msg("unmounting fuse filesystem...")
			mh.Unmount()
		}

		log.Info().Msg("exiting")
		os.Exit(1)
	}()

	fss, err := ts.Load()
	if err != nil {
		return fmt.Errorf("error when loading torrents: %w", err)
	}

	go func() {
		if mh == nil {
			return
		}

		if err := mh.Mount(fss); err != nil {
			log.Info().Err(err).Msg("error mounting filesystems")
		}
	}()

	go func() {
		if conf.Mounts.WebDAV.Enabled {
			port = webDAVPort
			if port == 0 {
				port = conf.Mounts.WebDAV.Port
			}

			cfs, err := fs.NewContainerFs(fss)
			if err != nil {
				log.Error().Err(err).Msg("error adding files to webDAV")
				return
			}

			if err := webdav.NewWebDAVServer(cfs, port, conf.Mounts.WebDAV.User, conf.Mounts.WebDAV.Pass); err != nil {
				log.Error().Err(err).Msg("error starting webDAV")
			}
		}

		log.Warn().Msg("webDAV configuration not found!")
	}()

	cfs, err := fs.NewContainerFs(fss)
	if err != nil {
		return fmt.Errorf("error when loading torrents: %w", err)
	}

	httpfs := httpfs.NewHTTPFS(cfs)
	logFilename := filepath.Join(conf.Log.Path, dlog.FileName)

	err = http.New(nil, ss, ts, conf, servers, httpfs, logFilename, conf)
	log.Error().Err(err).Msg("error initializing HTTP server")
	return err
}
