package asot

import (
	"fmt"
	"regexp"
	"strings"
)

var songIndexRG = regexp.MustCompile(`^INDEX.*?(\d+:\d+):\d*$`)

func formatTracklist(content string) string {
	type track struct {
		performer string
		title     string
		index     string
	}
	var tracks []*track
	var curTrack *track
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TRACK") {
			curTrack = &track{}
			tracks = append(tracks, curTrack)
		} else if strings.HasPrefix(line, "PERFORMER") {
			if curTrack != nil {
				curTrack.performer = cutTrackLine(line, "PERFORMER")
			}
		} else if strings.HasPrefix(line, "TITLE") {
			if curTrack != nil {
				curTrack.title = cutTrackLine(line, "TITLE")
			}
		} else if strings.HasPrefix(line, "INDEX") {
			curTrack.index = songIndexRG.ReplaceAllString(line, "$1")
		}
	}
	var buf strings.Builder
	for _, t := range tracks {
		buf.Grow(1024 * 10)
		buf.WriteString(fmt.Sprintf("%s\t%s — %s\n", t.index, t.performer, t.title))
	}
	return buf.String()
}

func cutTrackLine(line, prefix string) string {
	title := strings.TrimPrefix(line, prefix)
	title = strings.TrimPrefix(title, " ")
	title = strings.TrimPrefix(title, "\"")
	title = strings.TrimSuffix(title, "\"")
	return title
}
