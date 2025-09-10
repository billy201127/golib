package confuse

import (
	"sort"
)

// ============================================================================
// Obfuscator SDK - With Reversible Linear Congruential Mapping
// ============================================================================

type ObfuscatorSDK struct {
	dictionary []string
	seed       int
}

// NewObfuscatorSDK creates a new obfuscator SDK instance with embedded dictionary
func NewObfuscatorSDK(seed int) *ObfuscatorSDK {
	sdk := &ObfuscatorSDK{
		seed: seed,
	}
	sdk.loadEmbeddedDictionary()
	return sdk
}

// ObfuscateWord maps a word from the dictionary to another dictionary word (reversible)
func (sdk *ObfuscatorSDK) ObfuscateWord(word string) string {
	if len(sdk.dictionary) == 0 {
		return word
	}

	m := len(sdk.dictionary)

	// 确保种子为正数
	seed := sdk.seed
	if seed < 0 {
		seed = -seed
	}

	// 生成与m互质的乘法因子a
	a := sdk.generateCoprime(seed, m)
	b := seed % m

	// map word to dictionary index
	idx := sdk.wordToIndex(word)
	if idx < 0 {
		return word // not found
	}

	// apply linear congruential mapping
	newIdx := (a*idx + b) % m
	if newIdx < 0 {
		newIdx += m
	}
	return sdk.dictionary[newIdx]
}

func (sdk *ObfuscatorSDK) ObfuscateWords(words []string) map[string]string {
	obfWords := make(map[string]string)
	for _, word := range words {
		obfWords[word] = sdk.ObfuscateWord(word)
	}
	return obfWords
}

// DeobfuscateWords reverses ObfuscateWords mapping
func (sdk *ObfuscatorSDK) DeobfuscateWords(obfWords []string) map[string]string {
	words := make(map[string]string)
	for _, obfWord := range obfWords {
		words[obfWord] = sdk.DeobfuscateWord(obfWord)
	}
	return words
}

// DeobfuscateWord reverses ObfuscateWord mapping
func (sdk *ObfuscatorSDK) DeobfuscateWord(obfWord string) string {
	if len(sdk.dictionary) == 0 {
		return obfWord
	}

	m := len(sdk.dictionary)

	// 确保种子为正数
	seed := sdk.seed
	if seed < 0 {
		seed = -seed
	}

	// 生成与m互质的乘法因子a
	a := sdk.generateCoprime(seed, m)
	b := seed % m

	// find index of obfuscated word
	idx := sdk.wordToIndex(obfWord)
	if idx < 0 {
		return obfWord // not found
	}

	// compute modular inverse of a
	ainv := modularInverse(a, m)
	if ainv == -1 {
		return obfWord // cannot reverse
	}

	// reverse mapping: x = (y-b)*a^(-1) mod m
	origIdx := (ainv * ((idx - b + m) % m)) % m
	if origIdx < 0 {
		origIdx += m
	}
	return sdk.dictionary[origIdx]
}

// ============================================================================
// Helpers
// ============================================================================

// generateCoprime generates a number coprime to m using the seed
func (sdk *ObfuscatorSDK) generateCoprime(seed, m int) int {
	// 使用种子生成基础数
	base := seed % m
	if base <= 1 {
		base = 2
	}

	// 找到第一个与m互质的数
	for gcd(base, m) != 1 {
		base = (base + 1) % m
		if base <= 1 {
			base = 2
		}
	}

	return base
}

// gcd calculates the greatest common divisor
func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// wordToIndex returns the dictionary index of a word, or -1 if not found
func (sdk *ObfuscatorSDK) wordToIndex(word string) int {
	idx := sort.SearchStrings(sdk.dictionary, word)
	if idx >= len(sdk.dictionary) || sdk.dictionary[idx] != word {
		return -1
	}
	return idx
}

// modularInverse computes modular inverse of a under modulo m
func modularInverse(a, m int) int {
	if m == 1 {
		return 0
	}

	// 确保a为正数
	a = ((a % m) + m) % m

	t, newt := 0, 1
	r, newr := m, a

	for newr != 0 {
		quotient := r / newr
		t, newt = newt, t-quotient*newt
		r, newr = newr, r-quotient*newr
	}

	if r > 1 {
		return -1 // not invertible
	}
	if t < 0 {
		t += m
	}

	return t
}

// loadEmbeddedDictionary loads the built-in word dictionary
func (sdk *ObfuscatorSDK) loadEmbeddedDictionary() {
	sdk.dictionary = make([]string, len(Words))
	copy(sdk.dictionary, Words)
	sort.Strings(sdk.dictionary)
}
