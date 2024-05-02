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

var useTLS = flag.Bool("useTLS", true, "--useTLS=false")

//go:embed static
var staticDir embed.FS

func main() {
	flag.Parse()

	mux := setupMux()

	if *useTLS {
		go func() {
			log.Printf("Listening on :80 for redirects\n")
			if err := http.ListenAndServe(":80", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				targetUrl := url.URL{
					Scheme:   "https",
					Host:     r.Host,
					Path:     r.URL.Path,
					RawQuery: r.URL.RawQuery,
				}
				http.Redirect(rw, r, targetUrl.String(), http.StatusMovedPermanently)
			})); err != nil {
				log.Fatalf("https redirect ListenAndServe error: %+v", err)
			}
		}()
		log.Printf("Listening on :443\n")
		if err := http.ListenAndServeTLS(":443", "/etc/letsencrypt/live/mias.top/cert.pem", "/etc/letsencrypt/live/mias.top/privkey.pem", mux); err != nil {
			log.Fatalf("ListenAndServe error: %+v", err)
		}
	} else {
		log.Printf("Listening on :80\n")
		if err := http.ListenAndServe(":80", mux); err != nil {
			log.Fatalf("ListenAndServe error: %+v", err)
		}
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
