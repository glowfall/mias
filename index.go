package main

import (
	"fmt"
	"regexp"
	"strings"
)

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

func NewIndex() *index {
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
