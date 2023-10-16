package webdav

import (
	"fmt"
	"net/http"

	"git.kmsign.ru/royalcat/tstor/src/host/vfs"
	"github.com/rs/zerolog/log"
)

func NewWebDAVServer(fs vfs.Filesystem, port int, user, pass string) error {
	log.Info().Str("host", fmt.Sprintf("0.0.0.0:%d", port)).Msg("starting webDAV server")

	srv := newHandler(fs)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		username, password, _ := r.BasicAuth()
		if username == user && password == pass {
			srv.ServeHTTP(w, r)
			return
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="BASIC WebDAV REALM"`)
		w.WriteHeader(401)
		w.Write([]byte("401 Unauthorized\n"))
	})

	return http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), nil)
}
