package confuse

import (
	"sort"
	"strings"
)

// ============================================================================
// Obfuscator SDK - With Reversible Linear Congruential Mapping
// ============================================================================

// Character sets for position-dependent encryption
const (
	charsetLower = "abcdefghijklmnopqrstuvwxyz"
	charsetUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	charsetDigit = "0123456789"
)

type ObfuscatorSDK struct {
	dictionary       []string
	seed             int
	encryptOutOfDict bool // if true, encrypt out-of-dictionary words; if false, keep them unchanged
}

// NewObfuscatorSDK creates a new obfuscator SDK instance with embedded dictionary
// By default, out-of-dictionary words will be encrypted using character-level encryption
func NewObfuscatorSDK(seed int) *ObfuscatorSDK {
	sdk := &ObfuscatorSDK{
		seed:             seed,
		encryptOutOfDict: true, // default: encrypt out-of-dictionary words
	}
	sdk.loadEmbeddedDictionary()
	return sdk
}

// SetEncryptOutOfDict sets whether to encrypt out-of-dictionary words
// If set to false, out-of-dictionary words will be kept unchanged
func (sdk *ObfuscatorSDK) SetEncryptOutOfDict(encrypt bool) *ObfuscatorSDK {
	sdk.encryptOutOfDict = encrypt
	return sdk
}

// ObfuscateWord maps a word from the dictionary to another dictionary word (reversible)
// If word is not in dictionary and encryptOutOfDict is true, use character-level encryption
// If word is not in dictionary and encryptOutOfDict is false, return word unchanged
func (sdk *ObfuscatorSDK) ObfuscateWord(word string) string {
	if len(word) == 0 {
		return word
	}

	if len(sdk.dictionary) == 0 {
		if sdk.encryptOutOfDict {
			return sdk.encryptByChar(word)
		}
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
		// not found in dictionary
		if sdk.encryptOutOfDict {
			return sdk.encryptByChar(word)
		}
		return word // keep unchanged
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
// If word is not in dictionary and encryptOutOfDict is true, use character-level decryption
// If word is not in dictionary and encryptOutOfDict is false, return word unchanged
func (sdk *ObfuscatorSDK) DeobfuscateWord(obfWord string) string {
	if len(obfWord) == 0 {
		return obfWord
	}

	if len(sdk.dictionary) == 0 {
		if sdk.encryptOutOfDict {
			return sdk.decryptByChar(obfWord)
		}
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
		// not found in dictionary
		if sdk.encryptOutOfDict {
			return sdk.decryptByChar(obfWord)
		}
		return obfWord // keep unchanged
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

// ============================================================================
// Character-level Encryption (for out-of-dictionary words)
// ============================================================================

// encryptByChar encrypts a word using position-dependent character mapping
func (sdk *ObfuscatorSDK) encryptByChar(word string) string {
	result := make([]byte, len(word))
	for i := 0; i < len(word); i++ {
		result[i] = sdk.encryptChar(word[i], i)
	}
	return string(result)
}

// decryptByChar decrypts a word using position-dependent character mapping
func (sdk *ObfuscatorSDK) decryptByChar(word string) string {
	result := make([]byte, len(word))
	for i := 0; i < len(word); i++ {
		result[i] = sdk.decryptChar(word[i], i)
	}
	return string(result)
}

// encryptChar encrypts a single character at given position using LCG
func (sdk *ObfuscatorSDK) encryptChar(ch byte, pos int) byte {
	var charset string

	// determine character set based on character type
	if ch >= 'a' && ch <= 'z' {
		charset = charsetLower
	} else if ch >= 'A' && ch <= 'Z' {
		charset = charsetUpper
	} else if ch >= '0' && ch <= '9' {
		charset = charsetDigit
	} else {
		// non-alphanumeric characters remain unchanged
		return ch
	}

	m := len(charset)
	idx := strings.IndexByte(charset, ch)
	if idx < 0 {
		return ch
	}

	// 确保种子为正数
	seed := sdk.seed
	if seed < 0 {
		seed = -seed
	}

	// position-dependent LCG mapping
	a := sdk.generateCoprime(seed, m)
	b := (seed + pos) % m // each position has different offset

	newIdx := (a*idx + b) % m
	if newIdx < 0 {
		newIdx += m
	}

	return charset[newIdx]
}

// decryptChar decrypts a single character at given position using modular inverse
func (sdk *ObfuscatorSDK) decryptChar(ch byte, pos int) byte {
	var charset string

	// determine character set based on character type
	if ch >= 'a' && ch <= 'z' {
		charset = charsetLower
	} else if ch >= 'A' && ch <= 'Z' {
		charset = charsetUpper
	} else if ch >= '0' && ch <= '9' {
		charset = charsetDigit
	} else {
		// non-alphanumeric characters remain unchanged
		return ch
	}

	m := len(charset)
	idx := strings.IndexByte(charset, ch)
	if idx < 0 {
		return ch
	}

	// 确保种子为正数
	seed := sdk.seed
	if seed < 0 {
		seed = -seed
	}

	// position-dependent LCG mapping
	a := sdk.generateCoprime(seed, m)
	b := (seed + pos) % m
	ainv := modularInverse(a, m)

	if ainv == -1 {
		return ch // cannot reverse
	}

	// reverse mapping: x = (y-b)*a^(-1) mod m
	origIdx := (ainv * ((idx - b + m) % m)) % m
	if origIdx < 0 {
		origIdx += m
	}

	return charset[origIdx]
}
