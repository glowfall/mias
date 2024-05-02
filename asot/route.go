package asot

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"
)

//go:embed static
var staticFiles embed.FS

func Setup(mux *http.ServeMux) {
	index, err := NewIndexBuilder().BuildIndex()
	if err != nil {
		panic(err)
	}

	staticDir, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	mux.Handle("GET /asot/", http.StripPrefix("/asot/", http.FileServerFS(staticDir)))
	mux.HandleFunc("GET /asot/{$}", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(rw, r, staticFiles, "/static/index.html")
	})

	mux.HandleFunc("POST /asot/search", func(rw http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		songs := index.SearchSong(query)

		start := time.Now()
		fmt.Printf("search performed in %v, results: %v\n", time.Now().Sub(start), len(songs))

		rw.Header().Set("Content-Type", "application/json; charset=utf-8")

		if len(songs) == 0 {
			writeResults(rw, nil, 0)
		} else {
			results := make([]string, 0, len(songs))
			for _, song := range songs {
				results = append(results, fmt.Sprintf("ASOT %s: %s", song.episode, song.title))
			}
			writeResults(rw, results, len(results))
		}
	})
}
