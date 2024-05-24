package countdown

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func Setup(mux *http.ServeMux) {
	staticDir, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	mux.Handle("GET /countdown/", http.StripPrefix("/countdown/", http.FileServerFS(staticDir)))
	mux.HandleFunc("GET /countdown/{$}", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(rw, r, staticFiles, "/static/index.html")
	})
}
