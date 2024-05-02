package asot

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"
)

//go:embed static
var staticFiles embed.FS

func Setup(mux *http.ServeMux) {
	downloader := NewCachingDownloader()

	index, err := NewIndexBuilder(downloader).BuildIndex()
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
			return
		}
		results := make([]songResult, 0, len(songs))
		for _, song := range songs {
			results = append(results, songResult{
				Title:       fmt.Sprintf("ASOT %s: %s", song.episode, song.title),
				EpisodeHash: song.episodeHash,
			})
		}
		writeResults(rw, results, len(results))
	})
	mux.HandleFunc("GET /asot/tracklist", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/plain; charset=UTF-8")

		hash := r.URL.Query().Get("hash")
		content, err := downloader.GetCached(hash)
		if err != nil {
			_, err := rw.Write([]byte(fmt.Sprintf("Unable to load content by hash %s: %+v", hash, err)))
			if err != nil {
				panic(err)
			}
		}
		content = formatTracklist(content)
		_, err = rw.Write([]byte(content))
		if err != nil {
			panic(err)
		}
	})
}

type songResult struct {
	Title       string `json:"title"`
	EpisodeHash string `json:"episodeHash"`
}

func writeResults(rw http.ResponseWriter, results []songResult, count int) {
	result, err := json.Marshal(struct {
		Results []songResult `json:"results"`
		Count   int          `json:"count"`
	}{
		Results: results,
		Count:   count,
	})
	if err != nil {
		writeError(rw, "Unable to marshal result", err)
		return
	}

	if _, err := rw.Write(result); err != nil {
		writeError(rw, "Unable to write result", err)
		return
	}
}

func writeError(rw http.ResponseWriter, msg string, err error) {
	_, innerErr := rw.Write([]byte(fmt.Sprintf(`{"results":[{"title": "%s: %+v"}]}`, msg, err)))
	if innerErr != nil {
		fmt.Printf("Unable to write err: %+v", innerErr)
	}
	return
}
