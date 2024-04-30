package main

import (
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"
	"unicode/utf8"
)

//go:embed all:asot
var resourceFiles embed.FS

func main() {
	index, err := NewIndexBuilder().BuildIndex()
	if err != nil {
		panic(err)
	}

	mux := setupMux(index)

	const tls = true
	if tls {
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

func setupMux(index *index) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "index.html")
	})
	mux.Handle("GET /asot/", http.FileServerFS(resourceFiles))
	mux.HandleFunc("GET /asot/{$}", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(rw, r, resourceFiles, "/asot/index.html")
	})
	mux.HandleFunc("mias.top /", func(rw http.ResponseWriter, r *http.Request) {
		targetUrl := url.URL{
			Scheme:   "https",
			Host:     "www.mias.top",
			Path:     r.URL.Path,
			RawQuery: r.URL.RawQuery,
		}
		http.Redirect(rw, r, targetUrl.String(), http.StatusMovedPermanently)
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
	return mux
}

func writeResults(rw http.ResponseWriter, results []string, count int) {
	result, err := json.Marshal(struct {
		Results []string `json:"results"`
		Count   int      `json:"count"`
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
	_, innerErr := rw.Write([]byte(fmt.Sprintf(`{"results":"%s: %+v"}`, msg, err)))
	if innerErr != nil {
		fmt.Printf("Unable to write err: %+v", innerErr)
	}
	return
}

type indexBuilder struct {
	downloader *cachingDownloader
	index      *index
}

func NewIndexBuilder() *indexBuilder {
	return &indexBuilder{
		downloader: NewCachingDownloader(),
		index:      NewIndex(),
	}
}

// <a href="/download.php?type=cue&amp;folder=asot&amp;filename=Armin+van+Buuren+-+A+State+Of+Trance+1005+%28256+Kbps%29+baby967.cue"><img src="/layout/download.png" alt="Download!"></a>
var hrefRegexp = regexp.MustCompile(`<a href="(/download.php\?[^"]+)">`)

func (i *indexBuilder) BuildIndex() (*index, error) {
	body, err := i.downloader.DownloadOrGetCached("https://www.cuenation.com/?page=cues&folder=asot")
	if err != nil {
		return nil, err
	}

	submatches := hrefRegexp.FindAllStringSubmatch(string(body), -1)
	fmt.Printf("submatches: %v\n", len(submatches))

	for _, match := range submatches {
		if err := i.IndexCUE(html.UnescapeString(match[1])); err != nil {
			return nil, err
		}
	}

	return i.index, nil
}

var episodeRG = regexp.MustCompile(`(?im)^TITLE " *A State Of Trance (\d+)`)
var songRG = regexp.MustCompile(`(?m)^(?: +|  )?PERFORMER "?([^"]*?)"\r?\n(?: +|  )?TITLE "([^"]*?)"\r?\n(?: +|  )?INDEX.*?(\d+:\d+):\d*`)
var songReverseRG = regexp.MustCompile(`(?m)^(?: +|  )?TITLE "([^"]*?)"\r?\n(?: +|  )?PERFORMER "?([^"]*?)"\r?\n(?: +|  )?INDEX.*?(\d+:\d+):\d*`)

func cp1252ToUTF8(s string) string {
	var utf8Buf [utf8.UTFMax]byte
	bb := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		n := utf8.EncodeRune(utf8Buf[:], rune(s[i]))
		bb = append(bb, utf8Buf[:n]...)
	}
	return string(bb)
}

func (i *indexBuilder) IndexCUE(path string) error {
	link := "https://www.cuenation.com" + path
	bodyStr, err := i.downloader.DownloadOrGetCached(link)
	if err != nil {
		return err
	}
	if bodyStr == "The file doesn't exist!" {
		return nil
	}

	bodyStr = cp1252ToUTF8(bodyStr)

	titleSubmatches := episodeRG.FindAllStringSubmatch(bodyStr, -1)
	if len(titleSubmatches) == 0 {
		return fmt.Errorf("no title found in %s", link)
	}
	episode := titleSubmatches[0][1]

	songSubmatches := songRG.FindAllStringSubmatch(bodyStr, -1)
	if len(songSubmatches) == 0 {
		songSubmatches = songReverseRG.FindAllStringSubmatch(bodyStr, -1)
	}

	if len(songSubmatches) == 0 {
		return fmt.Errorf("no submatches")
	}

	for _, submatches := range songSubmatches {
		songPerformer := submatches[1]
		songTitle := submatches[2]
		songIndex := submatches[3]
		i.index.AddSong(episode, songPerformer, songTitle, songIndex)
	}

	return nil
}
