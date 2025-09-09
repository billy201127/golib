package confuse

import (
	"hash/fnv"
	"sort"
	"strings"
	"unicode"
)

// ============================================================================
// Obfuscator SDK - With Embedded Dictionary and Consistent Word Mapping
// ============================================================================

// ObfuscatorSDK provides a simple interface for field name obfuscation using embedded dictionary
type ObfuscatorSDK struct {
	dictionary []string
	seed       int
}

// ObfuscatorConfig provides configuration options for the obfuscator
type ObfuscatorConfig struct {
	Seed             int      // Fixed seed for deterministic results
	CustomDictionary []string // Optional custom dictionary words
}

// NewObfuscatorSDK creates a new obfuscator SDK instance with embedded dictionary
// seed: a fixed number to ensure deterministic obfuscation results
func NewObfuscatorSDK(seed int) *ObfuscatorSDK {
	sdk := &ObfuscatorSDK{
		seed: seed,
	}

	// Load embedded dictionary
	sdk.loadEmbeddedDictionary()

	return sdk
}

// NewObfuscatorSDKWithConfig creates an obfuscator with custom configuration
func NewObfuscatorSDKWithConfig(config ObfuscatorConfig) *ObfuscatorSDK {
	sdk := &ObfuscatorSDK{
		seed: config.Seed,
	}

	if len(config.CustomDictionary) > 0 {
		sdk.dictionary = make([]string, len(config.CustomDictionary))
		copy(sdk.dictionary, config.CustomDictionary)
		sort.Strings(sdk.dictionary)
	} else {
		sdk.loadEmbeddedDictionary()
	}

	return sdk
}

// ObfuscateFields takes a list of field names and returns their obfuscated mappings
// fields: slice of field names to obfuscate
// Returns: map[originalField]obfuscatedField
func (sdk *ObfuscatorSDK) ObfuscateFields(fields []string) map[string]string {
	if len(fields) == 0 {
		return make(map[string]string)
	}

	result := make(map[string]string)

	for _, field := range fields {
		if field == "" {
			result[field] = ""
			continue
		}

		words := sdk.splitFieldName(field)
		if len(words) == 0 {
			result[field] = field
			continue
		}

		obfuscatedWords := make([]string, len(words))
		for i, word := range words {
			obfuscatedWords[i] = sdk.obfuscateWord(word)
		}

		result[field] = strings.Join(obfuscatedWords, "_")
	}

	return result
}

// ObfuscateField obfuscates a single field name
// field: the field name to obfuscate
// Returns: obfuscated field name
func (sdk *ObfuscatorSDK) ObfuscateField(field string) string {
	mappings := sdk.ObfuscateFields([]string{field})
	if obfuscated, exists := mappings[field]; exists {
		return obfuscated
	}
	return field // fallback to original if obfuscation fails
}

// BatchObfuscate provides a convenient way to obfuscate multiple field groups
// fieldGroups: map where key is group name, value is slice of fields in that group
// Returns: map[groupName]map[originalField]obfuscatedField
func (sdk *ObfuscatorSDK) BatchObfuscate(fieldGroups map[string][]string) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for groupName, fields := range fieldGroups {
		result[groupName] = sdk.ObfuscateFields(fields)
	}

	return result
}

// GetReverseMapping creates a reverse mapping from obfuscated names back to original names
func (sdk *ObfuscatorSDK) GetReverseMapping(fields []string) map[string]string {
	forwardMapping := sdk.ObfuscateFields(fields)
	reverseMapping := make(map[string]string)

	for original, obfuscated := range forwardMapping {
		reverseMapping[obfuscated] = original
	}

	return reverseMapping
}

// GetDictionarySize returns the size of the current dictionary
func (sdk *ObfuscatorSDK) GetDictionarySize() int {
	return len(sdk.dictionary)
}

// SetSeed updates the seed for obfuscation (affects future obfuscations)
func (sdk *ObfuscatorSDK) SetSeed(seed int) {
	sdk.seed = seed
}

// ============================================================================
// Internal Implementation
// ============================================================================

// obfuscateWord maps a single word to an obfuscated word using deterministic hash
// This ensures the same word always maps to the same obfuscated word regardless of context
func (sdk *ObfuscatorSDK) obfuscateWord(word string) string {
	if len(sdk.dictionary) == 0 {
		return word
	}

	// Create a deterministic hash based on the word and seed
	h := fnv.New64a()
	h.Write([]byte(word))
	h.Write([]byte{byte(sdk.seed), byte(sdk.seed >> 8), byte(sdk.seed >> 16), byte(sdk.seed >> 24)})

	hash := h.Sum64()
	index := int(hash % uint64(len(sdk.dictionary)))

	return sdk.dictionary[index]
}

// splitFieldName splits a field name into words (camelCase and underscore separation)
func (sdk *ObfuscatorSDK) splitFieldName(fieldName string) []string {
	if fieldName == "" {
		return nil
	}

	// Split by underscores first
	parts := strings.Split(fieldName, "_")
	var words []string

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Split by camel case
		camelWords := sdk.splitCamelCase(part)
		for _, word := range camelWords {
			if word != "" {
				words = append(words, strings.ToLower(word))
			}
		}
	}

	return words
}

// splitCamelCase splits a camelCase string into words
func (sdk *ObfuscatorSDK) splitCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var words []string
	var current []rune

	for i, r := range s {
		if i == 0 {
			current = append(current, r)
			continue
		}

		// Split on uppercase letter following lowercase
		if unicode.IsUpper(r) && i > 0 && !unicode.IsUpper(rune(s[i-1])) {
			if len(current) > 0 {
				words = append(words, string(current))
				current = []rune{}
			}
		}
		current = append(current, r)
	}

	if len(current) > 0 {
		words = append(words, string(current))
	}

	return words
}

// loadEmbeddedDictionary loads the built-in word dictionary from embedded file
func (sdk *ObfuscatorSDK) loadEmbeddedDictionary() {
	// Use the embedded dictionary from the same package
	sdk.dictionary = make([]string, len(Words))
	copy(sdk.dictionary, Words)

	// Sort for deterministic ordering
	sort.Strings(sdk.dictionary)
}
