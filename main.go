package main

import (
	"crypto/sha1"
	"crypto/tls"
	"embed"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

//go:embed all:asot
var resourceFiles embed.FS

func main() {
	http.Handle("GET /asot/", http.FileServerFS(resourceFiles))
	http.HandleFunc("GET /asot/{$}", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(rw, r, resourceFiles, "/asot/index.html")
	})

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	index, err := crawl()
	if err != nil {
		panic(err)
	}

	http.HandleFunc("POST /asot/search", func(rw http.ResponseWriter, request *http.Request) {
		query := request.URL.Query().Get("query")
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

	fmt.Print("Listening on http://localhost:80\n")

	_ = http.ListenAndServe(":80", nil)
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

type Song struct {
	title     string
	signature string
	episode   string
	index     string
}

type trieNode struct {
	children map[rune]*trieNode
	songs    []*Song
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[rune]*trieNode),
	}
}

type index struct {
	trieRoot trieNode
}

func newIndex() *index {
	return &index{
		trieRoot: *newTrieNode(),
	}
}

var signatureRG = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func (i *index) AddSong(episode, performer, title, index string) {
	songPerformerLower := strings.ToLower(performer)
	songTitleLower := strings.ToLower(title)
	performerWords := strings.Fields(songPerformerLower)
	titleWords := strings.Fields(songTitleLower)

	song := &Song{
		title:     fmt.Sprintf("%s — %s (%s)", performer, title, index),
		signature: signatureRG.ReplaceAllString(songPerformerLower+songTitleLower, ""),
		episode:   episode,
		index:     index,
	}

	for _, words := range [][]string{performerWords, titleWords} {
		for _, word := range words {
			curNode := &i.trieRoot
			for i, char := range word {
				if curNode.children[char] == nil {
					curNode.children[char] = newTrieNode()
				}
				curNode = curNode.children[char]
				if i > 1 || len(word) < 3 {
					if containsSong(curNode, song) {
						continue
					}

					curNode.songs = append(curNode.songs, song)
				}
			}
		}
	}
}

func containsSong(curNode *trieNode, song *Song) bool {
	for _, curNodeSong := range curNode.songs {
		if curNodeSong.signature == song.signature {
			return true
		}
		if curNodeSong.episode == song.episode && curNodeSong.index == song.index {
			return true
		}
	}
	return false
}

func (i *index) SearchSong(song string) []*Song {
	var results []*Song
	words := strings.Fields(strings.ToLower(song))
	for _, word := range words {
		curNode := &i.trieRoot
		for _, char := range word {
			if curNode.children[char] == nil {
				curNode = nil
				break
			}
			curNode = curNode.children[char]
		}
		if curNode != nil {
			results = append(results, curNode.songs...)
		}
	}
	return results
}

// <a href="/download.php?type=cue&amp;folder=asot&amp;filename=Armin+van+Buuren+-+A+State+Of+Trance+1005+%28256+Kbps%29+baby967.cue"><img src="/layout/download.png" alt="Download!"></a>
var hrefRegexp = regexp.MustCompile(`<a href="(/download.php\?[^"]+)">`)

func crawl() (*index, error) {
	body, err := downloadOrGetCached("https://www.cuenation.com/?page=cues&folder=asot")
	if err != nil {
		return nil, err
	}

	submatches := hrefRegexp.FindAllStringSubmatch(string(body), -1)
	fmt.Printf("submatches: %v\n", len(submatches))

	index := newIndex()

	for _, match := range submatches {
		if err := loadCUE(html.UnescapeString(match[1]), index); err != nil {
			return nil, err
		}
	}

	return index, nil
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

func loadCUE(path string, index *index) error {
	link := "https://www.cuenation.com" + path
	bodyStr, err := downloadOrGetCached(link)
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

	for i := range songSubmatches {
		songPerformer := songSubmatches[i][1]
		songTitle := songSubmatches[i][2]
		songIndex := songSubmatches[i][3]
		index.AddSong(episode, songPerformer, songTitle, songIndex)
	}

	return nil
}

func downloadOrGetCached(link string) (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Dir(executablePath) + "/cache"

	if err := os.MkdirAll(cacheDir, 0777); err != nil {
		return "", err
	}

	cachedPath := cacheDir + "/" + getLinkHash(link)

	body, err := os.ReadFile(cachedPath)
	if err == nil {
		fmt.Printf("found in cache: %s\n", link)
		return string(body), nil
	}

	fmt.Printf("downloading: %s\n", link)

	r, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return "", err
	}
	r.Header.Add("Referer", "https://www.cuenation.com/?page=cues&folder=asot")
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(cachedPath, body, 0777); err != nil {
		fmt.Printf("unable to write to cache %s: %+v\n", cachedPath, err)
	}

	return string(body), nil
}

func getLinkHash(link string) string {
	sha := sha1.New()
	sha.Write([]byte(link))
	return hex.EncodeToString(sha.Sum(nil))
}
