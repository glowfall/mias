package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"net/url"

	"github.com/glowfall/asot/asot"
)

var listenAddress = flag.String("listenAddress", "127.0.0.1:8080", "HTTP listen address")

//go:embed static
var staticDir embed.FS

func main() {
	flag.Parse()

	mux := setupMux()

	log.Printf("Listening on http://%s/\n", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, mux); err != nil {
		log.Fatalf("ListenAndServe error: %+v", err)
	}
}

func setupMux() *http.ServeMux {
	staticFiles, err := fs.Sub(staticDir, "static")
	if err != nil {
		panic(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(rw, r, staticFiles, "index.html")
	})
	mux.Handle("GET /", http.FileServerFS(staticFiles))
	mux.HandleFunc("mias.top/", func(rw http.ResponseWriter, r *http.Request) {
		targetUrl := url.URL{
			Scheme:   "https",
			Host:     "www.mias.top",
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(rw, r, targetUrl.String(), http.StatusMovedPermanently)
	})

	asot.Setup(mux)
	return mux
}
