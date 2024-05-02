package asot

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type Song struct {
	title       string
	signature   string
	episode     string
	timeIndex   string
	episodeHash string
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

func NewIndex() *index {
	return &index{
		trieRoot: *newTrieNode(),
	}
}

var signatureRG = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func (idx *index) AddSong(episodeHash, episode, performer, title, timeIndex string) {
	songPerformerLower := strings.ToLower(performer)
	songTitleLower := strings.ToLower(title)
	performerWords := strings.Fields(songPerformerLower)
	titleWords := strings.Fields(songTitleLower)

	song := &Song{
		title:       fmt.Sprintf("%s — %s (%s)", performer, title, timeIndex),
		signature:   signatureRG.ReplaceAllString(songPerformerLower+songTitleLower, ""),
		episode:     episode,
		timeIndex:   timeIndex,
		episodeHash: episodeHash,
	}

	for _, words := range [][]string{performerWords, titleWords} {
		for _, word := range words {
			curNode := &idx.trieRoot
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
		if curNodeSong.episode != song.episode {
			continue
		}
		if curNodeSong.signature == song.signature {
			return true
		}
		if curNodeSong.timeIndex == song.timeIndex {
			return true
		}
	}
	return false
}

func (idx *index) SearchSong(song string) []*Song {
	var results []*Song
	words := strings.Fields(strings.ToLower(song))

	songRank := make(map[*Song]int)
	for _, word := range words {
		curNode := &idx.trieRoot
		for _, char := range word {
			if curNode.children[char] == nil {
				curNode = nil
				break
			}
			curNode = curNode.children[char]
		}
		if curNode != nil {
			for i := len(curNode.songs) - 1; i >= 0; i-- {
				song := curNode.songs[i]
				if rank, ok := songRank[song]; ok {
					songRank[song] = rank + 10000
					continue
				}
				songRank[song] = i
				results = append(results, song)
			}
		}
	}
	slices.SortFunc(results, func(a, b *Song) int {
		return songRank[b] - songRank[a]
	})
	return results
}
