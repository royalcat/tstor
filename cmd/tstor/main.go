package main

import (
	"fmt"

	"net"
	nethttp "net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"git.kmsign.ru/royalcat/tstor/src/config"
	"git.kmsign.ru/royalcat/tstor/src/host"
	"git.kmsign.ru/royalcat/tstor/src/host/repository"
	"git.kmsign.ru/royalcat/tstor/src/host/torrent"
	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	wnfs "github.com/willscott/go-nfs"

	"git.kmsign.ru/royalcat/tstor/src/export/fuse"
	"git.kmsign.ru/royalcat/tstor/src/export/httpfs"
	"git.kmsign.ru/royalcat/tstor/src/export/nfs"
	"git.kmsign.ru/royalcat/tstor/src/export/webdav"
	"git.kmsign.ru/royalcat/tstor/src/http"
	dlog "git.kmsign.ru/royalcat/tstor/src/log"
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
			return run(c.String(configFlag))
		},

		HideHelpCommand: true,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal().Err(err).Msg("problem starting application")
	}
}

func run(configPath string) error {

	conf, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	dlog.Load(&conf.Log)

	err = syscall.Setpriority(syscall.PRIO_PGRP, 0, 19)
	if err != nil {
		log.Err(err).Msg("set priority failed")
	}

	rep, err := repository.NewTorrentMetaRepository(conf.TorrentClient.MetadataFolder)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(conf.TorrentClient.MetadataFolder, 0744); err != nil {
		return fmt.Errorf("error creating metadata folder: %w", err)
	}

	fis, err := torrent.NewFileItemStore(filepath.Join(conf.TorrentClient.MetadataFolder, "items"), 2*time.Hour)
	if err != nil {
		return fmt.Errorf("error starting item store: %w", err)
	}
	defer fis.Close()

	id, err := torrent.GetOrCreatePeerID(filepath.Join(conf.TorrentClient.MetadataFolder, "ID"))
	if err != nil {
		return fmt.Errorf("error creating node ID: %w", err)
	}

	st, _, err := torrent.SetupStorage(conf.TorrentClient)
	if err != nil {
		return err
	}
	defer st.Close()

	c, err := torrent.NewClient(st, fis, &conf.TorrentClient, id)
	if err != nil {
		return fmt.Errorf("error starting torrent client: %w", err)
	}
	c.AddDhtNodes(conf.TorrentClient.DHTNodes)
	defer c.Close()

	ts := torrent.NewService(c, rep, conf.TorrentClient.AddTimeout, conf.TorrentClient.ReadTimeout)

	if err := os.MkdirAll(conf.DataFolder, 0744); err != nil {
		return fmt.Errorf("error creating data folder: %w", err)
	}
	cfs := host.NewStorage(conf.DataFolder, ts)

	if conf.Mounts.Fuse.Enabled {
		mh := fuse.NewHandler(conf.Mounts.Fuse.AllowOther, conf.Mounts.Fuse.Path)
		err := mh.Mount(cfs)
		if err != nil {
			return fmt.Errorf("mount fuse error: %w", err)
		}
		defer mh.Unmount()
	}

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
			err = nethttp.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", conf.Mounts.HttpFs.Port), nethttp.FileServer(httpfs))
			if err != nil {
				log.Error().Err(err).Msg("error starting HTTPFS")
			}
			// r := gin.New()

			// r.GET("*filepath", func(c *gin.Context) {
			// 	path := c.Param("filepath")
			// 	c.FileFromFS(path, httpfs)
			// })

			log.Info().Str("host", fmt.Sprintf("0.0.0.0:%d", conf.Mounts.HttpFs.Port)).Msg("starting HTTPFS")
			// if err := r.Run(fmt.Sprintf("0.0.0.0:%d", conf.Mounts.HttpFs.Port)); err != nil {
			// 	log.Error().Err(err).Msg("error starting HTTPFS")
			// }
		}()
	}

	if conf.Mounts.NFS.Enabled {
		go func() {
			listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", conf.Mounts.NFS.Port))
			panicOnErr(err, "starting TCP listener")
			log.Info().Str("host", listener.Addr().String()).Msg("starting NFS server")
			handler, err := nfs.NewNFSv3Handler(cfs)
			panicOnErr(err, "creating NFS handler")
			panicOnErr(wnfs.Serve(listener, handler), "serving nfs")
		}()
	}

	dataFS := vfs.NewOsFs(conf.DataFolder)

	go func() {
		if err := webdav.NewWebDAVServer(dataFS, 36912, conf.Mounts.WebDAV.User, conf.Mounts.WebDAV.Pass); err != nil {
			log.Error().Err(err).Msg("error starting webDAV")
		}

		log.Warn().Msg("webDAV configuration not found!")
	}()

	go func() {
		logFilename := filepath.Join(conf.Log.Path, dlog.FileName)

		err = http.New(nil, nil, ts, logFilename, conf)
		log.Error().Err(err).Msg("error initializing HTTP server")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	return nil
}

func panicOnErr(err error, desc string) {
	if err == nil {
		return
	}
	log.Err(err).Msg(desc)
	log.Panic()
}
