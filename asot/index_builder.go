package asot

import (
	"errors"
	"fmt"
	"html"
	"log"
	"regexp"
	"sync"
	"unicode/utf8"
)

type indexBuilder struct {
	downloader  *cachingDownloader
	index       *index
	onceBuilder sync.Once
}

func NewIndexBuilder(downloader *cachingDownloader) *indexBuilder {
	return &indexBuilder{
		downloader: downloader,
		index:      NewIndex(),
	}
}

// <a href="/download.php?type=cue&amp;folder=asot&amp;filename=Armin+van+Buuren+-+A+State+Of+Trance+1005+%28256+Kbps%29+baby967.cue"><img src="/layout/download.png" alt="Download!"></a>
var hrefRegexp = regexp.MustCompile(`<a href="(/download.php\?[^"]+)">`)

func (i *indexBuilder) BuildIndexAsync() *index {
	i.onceBuilder.Do(func() {
		go i.buildIndex()
	})

	return i.index
}

func (i *indexBuilder) buildIndex() {
	log.Println("Starting index build in background...")

	body, err := i.downloader.DownloadOrGetCached("https://www.cuenation.com/?page=cues&folder=asot")
	if err != nil {
		log.Printf("Error downloading index page: %+v", err)
		return
	}

	submatches := hrefRegexp.FindAllStringSubmatch(body, -1)
	log.Printf("Found %d episodes to index\n", len(submatches))

	ch := make(chan func() error, 100)

	resCh := make(chan error, 100)
	var errs []error

	go func() {
		for err := range resCh {
			if err != nil {
				errs = append(errs, err)
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					resCh <- fmt.Errorf("panic recovered: %v", r)
				}
			}()
			for f := range ch {
				resCh <- f()
			}
		}()
	}

	for _, match := range submatches {
		ch <- func() error {
			return i.IndexCUE(html.UnescapeString(match[1]))
		}
	}
	close(ch)
	wg.Wait()
	close(resCh)

	if len(errs) > 0 {
		log.Printf("Index build completed with errors: %+v", errors.Join(errs...))
	} else {
		log.Println("Index build completed successfully")
	}
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

	titleSubmatches := episodeRG.FindAllStringSubmatch(bodyStr, -1)
	if len(titleSubmatches) == 0 {
		return fmt.Errorf("no title found in %s", link)
	}
	episode := titleSubmatches[0][1]
	episodeHash := i.downloader.LinkHash(link)

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
		i.index.AddSong(episodeHash, episode, songPerformer, songTitle, songTimeIndex)
	}

	return nil
}

func (i *indexBuilder) TracklistByHash(hash string) (string, error) {
	return i.downloader.GetCached(hash)
}
