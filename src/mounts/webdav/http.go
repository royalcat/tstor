package webdav

import (
	"fmt"
	"net/http"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/rs/zerolog/log"
)

func NewWebDAVServer(fs vfs.Filesystem, port int, user, pass string) error {

	srv := newHandler(fs)

	serveMux := http.NewServeMux()

	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		username, password, _ := r.BasicAuth()
		if username == user && password == pass {
			srv.ServeHTTP(w, r)
			return
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="BASIC WebDAV REALM"`)
		w.WriteHeader(401)
		_, _ = w.Write([]byte("401 Unauthorized\n"))
	})

	//nolint:exhaustruct
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", port),
		Handler: serveMux,
	}

	log.Info().Str("host", httpServer.Addr).Msg("starting webDAV server")

	return httpServer.ListenAndServe()
}
