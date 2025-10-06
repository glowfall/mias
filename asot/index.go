package asot

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
)

type Song struct {
	title       string
	signature   string
	episodeNum  int
	timeIndex   string
	episodeHash string
}

func (s *Song) less(s2 *Song) bool {
	if s.signature != s2.signature {
		return s.signature < s2.signature
	}
	return s.timeIndex < s2.timeIndex
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
	lock     sync.RWMutex
}

func NewIndex() *index {
	return &index{
		trieRoot: *newTrieNode(),
	}
}

var signatureRG = regexp.MustCompile(`[^a-zA-Z0-9]+`)
var cleanerRG = regexp.MustCompile(`[^[:alpha:]\s'’ʼ-]+`)

func cleanString(str string) string {
	str = cleanerRG.ReplaceAllLiteralString(str, "")
	str = strings.ToLower(str)
	return str
}

func (idx *index) AddSong(episodeHash, episode, performer, title, timeIndex string) {
	songPerformerLower := cleanString(performer)
	songTitleLower := cleanString(title)
	performerWords := strings.Fields(songPerformerLower)
	titleWords := strings.Fields(songTitleLower)

	episodeNum, _ := strconv.Atoi(episode)

	song := &Song{
		title:       fmt.Sprintf("%s — %s (%s)", performer, title, timeIndex),
		signature:   signatureRG.ReplaceAllString(songPerformerLower+songTitleLower, ""),
		episodeNum:  episodeNum,
		timeIndex:   timeIndex,
		episodeHash: episodeHash,
	}

	idx.lock.Lock()
	defer idx.lock.Unlock()

	// Track which nodes already have this song to avoid duplicates
	addedNodes := make(map[*trieNode]bool)
	for _, words := range [][]string{performerWords, titleWords} {
		for _, word := range words {
			curNode := &idx.trieRoot
			for i, char := range word {
				if curNode.children[char] == nil {
					curNode.children[char] = newTrieNode()
				}
				curNode = curNode.children[char]
				if (i > 1 || len(word) < 3) && !addedNodes[curNode] {
					curNode.songs = append(curNode.songs, song)
					addedNodes[curNode] = true
				}
			}
		}
	}
}

func (idx *index) SearchSong(song string) []*Song {
	words := strings.Fields(strings.ToLower(song))

	idx.lock.RLock()
	defer idx.lock.RUnlock()

	songMatchCount := make(map[*Song]int)
	for _, word := range words {
		curNode := &idx.trieRoot
		for _, char := range word {
			children := curNode.children[char]
			if children == nil {
				curNode = nil
				break
			}
			curNode = children
		}
		if curNode != nil {
			for _, song := range curNode.songs {
				songMatchCount[song]++
			}
		}
	}

	results := make([]*Song, 0, len(songMatchCount))
	for song := range songMatchCount {
		results = append(results, song)
	}
	results = dedupOR(results)

	slices.SortFunc(results, func(a, b *Song) int {
		matchA := songMatchCount[a]
		matchB := songMatchCount[b]
		if matchB != matchA {
			return matchB - matchA
		}
		if a.episodeNum != b.episodeNum {
			return a.episodeNum - b.episodeNum
		}
		if a.signature != b.signature {
			return cmp.Compare(a.signature, b.signature)
		}
		if a.timeIndex != b.timeIndex {
			return cmp.Compare(a.timeIndex, b.timeIndex)
		}
		return 0
	})
	return results
}

// We retain only the one song from group of the same time code or signature in every episode
func dedupOR(songs []*Song) []*Song {
	if len(songs) == 0 {
		return nil
	}

	type Key struct {
		episode int
		str     string
	}
	bySig := make(map[Key][]int, len(songs))
	bySlot := make(map[Key][]int, len(songs))
	for i, s := range songs {
		bySig[Key{s.episodeNum, s.signature}] = append(bySig[Key{s.episodeNum, s.signature}], i)
		bySlot[Key{s.episodeNum, s.timeIndex}] = append(bySlot[Key{s.episodeNum, s.timeIndex}], i)
	}

	vis := make([]bool, len(songs))
	res := make([]*Song, 0, len(songs))

	for i := 0; i < len(songs); i++ {
		if vis[i] {
			continue
		}

		q := []int{i}
		vis[i] = true
		best := songs[i]

		for len(q) > 0 {
			v := q[0]
			q = q[1:]

			visit := func(nb int) {
				if vis[nb] {
					return
				}
				vis[nb] = true
				q = append(q, nb)
				if songs[nb].less(best) {
					best = songs[nb]
				}
			}
			for _, nb := range bySig[Key{songs[v].episodeNum, songs[v].signature}] {
				visit(nb)
			}
			for _, nb := range bySlot[Key{songs[v].episodeNum, songs[v].timeIndex}] {
				visit(nb)
			}
		}

		res = append(res, best)
	}

	return res
}
