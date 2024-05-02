package asot

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"unicode/utf8"
)

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
		songTimeIndex := submatches[3]
		i.index.AddSong(episode, songPerformer, songTitle, songTimeIndex)
	}

	return nil
}
