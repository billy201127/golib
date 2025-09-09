package confuse

import (
	_ "embed"
	"strings"
)

//go:embed data/words.txt
var wordsFile string

var (
	Words   []string
	WordSet map[string]struct{}
)

// load the embedded dictionary
func init() {
	// one word per line
	Words = strings.Split(strings.TrimSpace(wordsFile), "\n")

	WordSet = make(map[string]struct{}, len(Words))
	for _, w := range Words {
		WordSet[w] = struct{}{}
	}
}

// GetWords returns all words from the embedded dictionary
func GetWords() []string {
	return Words
}

// HasWord checks if a word exists in the dictionary
func HasWord(word string) bool {
	_, exists := WordSet[word]
	return exists
}

// GetWordCount returns the total number of words in the dictionary
func GetWordCount() int {
	return len(Words)
}
