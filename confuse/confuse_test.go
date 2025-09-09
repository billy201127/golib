package confuse

import (
	"testing"
)

// TestNewObfuscatorSDK tests the basic constructor
func TestNewObfuscatorSDK(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)
	if sdk == nil {
		t.Fatal("Expected non-nil SDK")
	}

	// Test dictionary is loaded
	if sdk.GetDictionarySize() == 0 {
		t.Error("Expected dictionary to be loaded")
	}

	t.Logf("Dictionary size: %d", sdk.GetDictionarySize())
}

// TestNewObfuscatorSDKWithConfig tests the config-based constructor
func TestNewObfuscatorSDKWithConfig(t *testing.T) {
	config := ObfuscatorConfig{
		Seed:             54321,
		CustomDictionary: []string{"custom", "words", "for", "testing"},
	}

	sdk := NewObfuscatorSDKWithConfig(config)
	if sdk == nil {
		t.Fatal("Expected non-nil SDK")
	}

	// Should have custom dictionary + embedded dictionary
	dictSize := sdk.GetDictionarySize()
	if dictSize < 4 {
		t.Errorf("Expected dictionary size >= 4, got %d", dictSize)
	}

	t.Logf("Dictionary size with custom words: %d", dictSize)
}

// TestObfuscateField tests single field obfuscation
func TestObfuscateField(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	testCases := []struct {
		name     string
		field    string
		expected string // We'll check deterministic behavior
	}{
		{"Simple field", "username", ""},
		{"CamelCase field", "UserName", ""},
		{"Snake case", "user_id", ""},
		{"Mixed case", "EmailAddress", ""},
		{"Single word", "name", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sdk.ObfuscateField(tc.field)
			if result == "" {
				t.Errorf("Expected non-empty result for field %s", tc.field)
			}
			if result == tc.field {
				t.Errorf("Expected obfuscated result to be different from original field %s", tc.field)
			}

			// Test deterministic behavior
			result2 := sdk.ObfuscateField(tc.field)
			if result != result2 {
				t.Errorf("Expected deterministic results, got %s and %s", result, result2)
			}

			t.Logf("Field: %s -> %s", tc.field, result)
		})
	}
}

// TestObfuscateFields tests batch field obfuscation
func TestObfuscateFields(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	fields := []string{"username", "password", "email", "UserID", "created_at"}

	result := sdk.ObfuscateFields(fields)
	if len(result) != len(fields) {
		t.Errorf("Expected %d results, got %d", len(fields), len(result))
	}

	// Check all fields are mapped
	for _, field := range fields {
		obfuscated, exists := result[field]
		if !exists {
			t.Errorf("Expected field %s to be in result", field)
		}
		if obfuscated == "" {
			t.Errorf("Expected non-empty obfuscated value for field %s", field)
		}
		if obfuscated == field {
			t.Errorf("Expected obfuscated value to be different from original for field %s", field)
		}
		t.Logf("Field: %s -> %s", field, obfuscated)
	}
}

// TestBatchObfuscate tests batch obfuscation with groups
func TestBatchObfuscate(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	fieldGroups := map[string][]string{
		"user_table":    {"user_id", "username", "email"},
		"product_table": {"product_id", "product_name", "price"},
		"order_table":   {"order_id", "user_id", "product_id", "created_at"},
	}

	result := sdk.BatchObfuscate(fieldGroups)

	if len(result) != len(fieldGroups) {
		t.Errorf("Expected %d groups in result, got %d", len(fieldGroups), len(result))
	}

	for groupName, fields := range fieldGroups {
		groupResult, exists := result[groupName]
		if !exists {
			t.Errorf("Expected group %s to be in result", groupName)
			continue
		}

		if len(groupResult) != len(fields) {
			t.Errorf("Expected %d fields in group %s, got %d", len(fields), groupName, len(groupResult))
		}

		t.Logf("Group: %s", groupName)
		for _, field := range fields {
			obfuscated, exists := groupResult[field]
			if !exists {
				t.Errorf("Expected field %s to be in group %s result", field, groupName)
			} else {
				t.Logf("  %s -> %s", field, obfuscated)
			}
		}
	}
}

// TestCreateReverseMapping tests reverse mapping creation
func TestCreateReverseMapping(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	fields := []string{"username", "password", "email"}

	// First get the forward mapping
	forwardMap := sdk.ObfuscateFields(fields)

	// Then get the reverse mapping
	reverseMap := sdk.GetReverseMapping(fields)

	if len(reverseMap) != len(forwardMap) {
		t.Errorf("Expected reverse map size %d, got %d", len(forwardMap), len(reverseMap))
	}

	// Check that reverse mapping is actually reverse
	for original, obfuscated := range forwardMap {
		reversedOriginal, exists := reverseMap[obfuscated]
		if !exists {
			t.Errorf("Expected obfuscated field %s to be in reverse map", obfuscated)
		}
		if reversedOriginal != original {
			t.Errorf("Expected reverse mapping of %s to be %s, got %s", obfuscated, original, reversedOriginal)
		}
		t.Logf("Reverse: %s -> %s", obfuscated, reversedOriginal)
	}
}

// TestDeterministicBehavior tests that same seed produces same results
func TestDeterministicBehavior(t *testing.T) {
	seed := 98765
	fields := []string{"UserName", "EmailAddress", "user_id", "created_at"}

	// Create two SDK instances with same seed
	sdk1 := NewObfuscatorSDK(seed)
	sdk2 := NewObfuscatorSDK(seed)

	result1 := sdk1.ObfuscateFields(fields)
	result2 := sdk2.ObfuscateFields(fields)

	// Results should be identical
	for _, field := range fields {
		if result1[field] != result2[field] {
			t.Errorf("Expected deterministic results for field %s, got %s and %s",
				field, result1[field], result2[field])
		}
	}

	t.Log("Deterministic behavior verified")
}

// TestDifferentSeeds tests that different seeds produce different results
func TestDifferentSeeds(t *testing.T) {
	fields := []string{"username", "password"}

	sdk1 := NewObfuscatorSDK(111)
	sdk2 := NewObfuscatorSDK(222)

	result1 := sdk1.ObfuscateFields(fields)
	result2 := sdk2.ObfuscateFields(fields)

	// Results should be different (at least for some fields)
	differentCount := 0
	for _, field := range fields {
		if result1[field] != result2[field] {
			differentCount++
			t.Logf("Field %s: seed 111 -> %s, seed 222 -> %s",
				field, result1[field], result2[field])
		}
	}

	if differentCount == 0 {
		t.Error("Expected different seeds to produce different results")
	}
}

// TestSetSeed tests seed modification
func TestSetSeed(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)
	field := "username"

	result1 := sdk.ObfuscateField(field)

	// Change seed
	sdk.SetSeed(54321)
	result2 := sdk.ObfuscateField(field)

	if result1 == result2 {
		t.Error("Expected different results after changing seed")
	}

	t.Logf("Field %s: seed 12345 -> %s, seed 54321 -> %s", field, result1, result2)
}

// TestEmptyAndEdgeCases tests edge cases
func TestEmptyAndEdgeCases(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	testCases := []struct {
		name  string
		field string
	}{
		{"Empty string", ""},
		{"Single character", "a"},
		{"Numbers", "123"},
		{"Special characters", "field_with_123_numbers"},
		{"Long field name", "veryLongFieldNameWithManyCamelCaseWords"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sdk.ObfuscateField(tc.field)
			// Should not panic and should return some result
			t.Logf("Field: '%s' -> '%s'", tc.field, result)
		})
	}
}

// TestCustomDictionary tests custom dictionary functionality
func TestCustomDictionary(t *testing.T) {
	customWords := []string{"apple", "banana", "cherry", "date"}

	config := ObfuscatorConfig{
		Seed:             12345,
		CustomDictionary: customWords,
	}

	sdk := NewObfuscatorSDKWithConfig(config)

	// Dictionary should include custom words
	dictSize := sdk.GetDictionarySize()
	if dictSize < len(customWords) {
		t.Errorf("Expected dictionary size >= %d, got %d", len(customWords), dictSize)
	}

	// Test obfuscation still works
	result := sdk.ObfuscateField("testField")
	if result == "" {
		t.Error("Expected non-empty obfuscation result")
	}

	t.Logf("Custom dictionary size: %d, obfuscation result: %s", dictSize, result)
}

// BenchmarkObfuscateField benchmarks single field obfuscation
func BenchmarkObfuscateField(b *testing.B) {
	sdk := NewObfuscatorSDK(12345)
	field := "UserName"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sdk.ObfuscateField(field)
	}
}

// BenchmarkObfuscateFields benchmarks batch field obfuscation
func BenchmarkObfuscateFields(b *testing.B) {
	sdk := NewObfuscatorSDK(12345)
	fields := []string{"username", "password", "email", "UserID", "created_at", "updated_at", "first_name", "last_name"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sdk.ObfuscateFields(fields)
	}
}

// TestWordMappingConsistency tests if the same word always maps to the same obfuscated word
func TestWordMappingConsistency(t *testing.T) {
	seed := 12345
	targetWord := "user"

	// Different field sets that all contain the word "user"
	fieldSets := [][]string{
		{"user_id"},
		{"username"},
		{"user_id", "username"},
		{"user_id", "username", "user_profile"},
		{"username", "user_profile", "user_settings"},
		{"user_email", "user_phone", "user_address"},
	}

	// Track obfuscated versions of "user" across different field sets
	var allObfuscatedVersions []string

	for i, fields := range fieldSets {
		sdk := NewObfuscatorSDK(seed)
		mappings := sdk.ObfuscateFields(fields)

		t.Logf("Field set %d: %v", i+1, fields)
		t.Logf("Mappings: %v", mappings)

		// Find the obfuscated version of "user" in this field set
		for _, field := range fields {
			obfuscated := mappings[field]
			originalWords := sdk.splitFieldName(field)
			obfuscatedWords := sdk.splitFieldName(obfuscated)

			for j, origWord := range originalWords {
				if origWord == targetWord && j < len(obfuscatedWords) {
					allObfuscatedVersions = append(allObfuscatedVersions, obfuscatedWords[j])
					t.Logf("  Found '%s' -> '%s' in field '%s'", targetWord, obfuscatedWords[j], field)
					break
				}
			}
		}
	}

	t.Logf("\nAll obfuscated versions of '%s': %v", targetWord, allObfuscatedVersions)

	// Check consistency - all versions should be identical
	if len(allObfuscatedVersions) > 1 {
		firstVersion := allObfuscatedVersions[0]
		for i, version := range allObfuscatedVersions {
			if version != firstVersion {
				t.Errorf("INCONSISTENT: Field set produced different mapping for '%s': expected '%s', got '%s' at position %d",
					targetWord, firstVersion, version, i)
			}
		}

		if t.Failed() {
			t.Logf("PROBLEM: The same word maps to different obfuscated words depending on the field set!")
		} else {
			t.Logf("SUCCESS: Word '%s' consistently maps to '%s' across all field sets", targetWord, firstVersion)
		}
	}
}

// TestMultipleWordConsistency tests consistency for multiple words
func TestMultipleWordConsistency(t *testing.T) {
	seed := 12345
	testWords := []string{"user", "name", "id", "email", "password"}

	for _, word := range testWords {
		t.Run("word_"+word, func(t *testing.T) {
			// Test the same word in different contexts
			fieldSets := [][]string{
				{word},
				{word + "_id"},
				{word + "_name"},
				{"first_" + word, "last_" + word},
				{"old_" + word, "new_" + word, "temp_" + word},
			}

			var allMappings []string

			for _, fields := range fieldSets {
				sdk := NewObfuscatorSDK(seed)
				mappings := sdk.ObfuscateFields(fields)

				for _, field := range fields {
					obfuscated := mappings[field]
					originalWords := sdk.splitFieldName(field)
					obfuscatedWords := sdk.splitFieldName(obfuscated)

					for j, origWord := range originalWords {
						if origWord == word && j < len(obfuscatedWords) {
							allMappings = append(allMappings, obfuscatedWords[j])
							break
						}
					}
				}
			}

			// All mappings should be identical
			if len(allMappings) > 1 {
				firstMapping := allMappings[0]
				for i, mapping := range allMappings {
					if mapping != firstMapping {
						t.Errorf("Inconsistent mapping for word '%s': expected '%s', got '%s' at position %d",
							word, firstMapping, mapping, i)
					}
				}

				if !t.Failed() {
					t.Logf("SUCCESS: Word '%s' consistently maps to '%s'", word, firstMapping)
				}
			}
		})
	}
}
