package main

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

func TestIndex(t *testing.T) {
	index := newIndex()
	index.AddSong("300", "Armin", "Song1", "10:00")
	index.AddSong("300", "Mat Zoo", "Clockwork", "01:00")
	index.AddSong("306", "Mat Zoo", "Song1", "01:00")
	index.AddSong("302", "Armin", "Clockwork", "10:00")

	println(spew.Sdump(index.SearchSong("cLock")))
	println(spew.Sdump(index.SearchSong("zoo")))
}

const content1 = `PERFORMER "Armin van Buuren"
TITLE "A State Of Trance 1170 (2024-04-25) (TOP 1000 2024: Final 50) [MM] (SBD)"
FILE "A_State_Of_Trance_1170-SBD_(25-04-2024).mp3" MP3
  TRACK 01 AUDIO
    PERFORMER "ASOT"
    TITLE "Intro"
    INDEX 01 00:00:00
  TRACK 02 AUDIO
    PERFORMER "[#50] Bryan Kearney & Plumb"
    TITLE "All Over Again"
    INDEX 01 03:44:45
`

const content2 = `PERFORMER "Armin van Buuren"
TITLE "A State of Trance 1169 (2024-04-18) [IYPP]"
FILE "Armin van Buuren - A State Of Trance 1169 (256 Kbps) baby967.mp3" MP3
  TRACK 01 AUDIO
    PERFORMER "A State Of Trance"
    TITLE "Intro"
    INDEX 01 00:00:00
  TRACK 02 AUDIO
    PERFORMER "AOA"
    TITLE "Burn (The Rise)"
    INDEX 01 00:41:27
  TRACK 03 AUDIO
    PERFORMER "Steve Brian"
    TITLE "The Observer"
    INDEX 01 04:12:03
`

func TestTitleRG(t *testing.T) {
	for _, content := range []string{content1, content2} {
		titleSubmatches := episodeRG.FindAllStringSubmatch(content, -1)
		require.Len(t, titleSubmatches, 1)

		songPerformerSubmatches := songRG.FindAllStringSubmatch(content, -1)
		require.Len(t, songPerformerSubmatches, 52)
	}
}
