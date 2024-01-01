package http

import (
	"bytes"
	"io"
	"math"
	"net/http"
	"os"

	"git.kmsign.ru/royalcat/tstor/src/host/service"
	"github.com/anacrolix/missinggo/v2/filecache"
	"github.com/gin-gonic/gin"
)

var apiStatusHandler = func(fc *filecache.Cache, ss *service.Stats) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		stat := gin.H{
			"torrentStats": ss.GlobalStats(),
		}

		if fc != nil {
			stat["cacheItems"] = fc.Info().NumItems
			stat["cacheFilled"] = fc.Info().Filled / 1024 / 1024
			stat["cacheCapacity"] = fc.Info().Capacity / 1024 / 1024
		}

		// TODO move to a struct
		ctx.JSON(http.StatusOK, stat)
	}
}

// var apiServersHandler = func(ss []*service.Server) gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		var infos []*torrent.ServerInfo
// 		for _, s := range ss {
// 			infos = append(infos, s.Info())
// 		}
// 		ctx.JSON(http.StatusOK, infos)
// 	}
// }

// var apiRoutesHandler = func(ss *service.Stats) gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		s := ss.RoutesStats()
// 		sort.Sort(torrent.ByName(s))
// 		ctx.JSON(http.StatusOK, s)
// 	}
// }

// var apiAddTorrentHandler = func(s *service.Service) gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		route := ctx.Param("route")

// 		var json RouteAdd
// 		if err := ctx.ShouldBindJSON(&json); err != nil {
// 			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		if err := s.AddMagnet(route, json.Magnet); err != nil {
// 			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		ctx.JSON(http.StatusOK, nil)
// 	}
// }

// var apiDelTorrentHandler = func(s *service.Service) gin.HandlerFunc {
// 	return func(ctx *gin.Context) {
// 		route := ctx.Param("route")
// 		hash := ctx.Param("torrent_hash")

// 		if err := s.RemoveFromHash(route, hash); err != nil {
// 			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			return
// 		}

// 		ctx.JSON(http.StatusOK, nil)
// 	}
// }

var apiLogHandler = func(path string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		f, err := os.Open(path)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fi, err := f.Stat()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		max := math.Max(float64(-fi.Size()), -1024*8*8)
		_, err = f.Seek(int64(max), io.SeekEnd)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var b bytes.Buffer
		ctx.Stream(func(w io.Writer) bool {
			_, err := b.ReadFrom(f)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return false
			}

			_, err = b.WriteTo(w)
			if err != nil {
				ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return false
			}

			return true
		})

		if err := f.Close(); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
}
