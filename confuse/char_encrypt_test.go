package confuse

import (
	"testing"
)

// TestCharacterEncryption tests character-level encryption for out-of-dictionary words
func TestCharacterEncryption(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	testCases := []struct {
		name     string
		input    string
		expected string // we'll verify reversibility instead of exact match
	}{
		{
			name:  "lowercase letters",
			input: "xyzabc", // unlikely to be in dictionary
		},
		{
			name:  "uppercase letters",
			input: "XYZABC",
		},
		{
			name:  "mixed case",
			input: "XyzAbc",
		},
		{
			name:  "with digits",
			input: "test123xyz", // unlikely to be in dictionary
		},
		{
			name:  "with special chars",
			input: "test_word_123",
		},
		{
			name:  "email address",
			input: "user123@test.com",
		},
		{
			name:  "mixed all",
			input: "User123_Test",
		},
		{
			name:  "chinese characters",
			input: "你好world123",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt
			encrypted := sdk.ObfuscateWord(tc.input)
			t.Logf("Input: %s -> Encrypted: %s", tc.input, encrypted)

			// Verify length is preserved
			if len(encrypted) != len(tc.input) {
				t.Errorf("Length mismatch: expected %d, got %d", len(tc.input), len(encrypted))
			}

			// Decrypt
			decrypted := sdk.DeobfuscateWord(encrypted)
			t.Logf("Encrypted: %s -> Decrypted: %s", encrypted, decrypted)

			// Verify reversibility
			if decrypted != tc.input {
				t.Errorf("Reversibility failed: expected %s, got %s", tc.input, decrypted)
			}

			// Verify encryption actually changed the value (for alphanumeric inputs)
			hasAlphanumeric := false
			for _, ch := range tc.input {
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
					hasAlphanumeric = true
					break
				}
			}
			if hasAlphanumeric && encrypted == tc.input {
				t.Errorf("Encryption did not change the input: %s", tc.input)
			}
		})
	}
}

// TestDictionaryWordStillWorks tests that dictionary words still use LCG mapping
func TestDictionaryWordStillWorks(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	// Test with a word that should be in the dictionary
	// (assuming "the" is a common word in the dictionary)
	word := "the"

	encrypted := sdk.ObfuscateWord(word)
	t.Logf("Dictionary word '%s' -> '%s'", word, encrypted)

	decrypted := sdk.DeobfuscateWord(encrypted)
	t.Logf("Decrypted: '%s'", decrypted)

	if decrypted != word {
		t.Errorf("Dictionary word reversibility failed: expected %s, got %s", word, decrypted)
	}

	// The encrypted result should also be a dictionary word
	if sdk.wordToIndex(encrypted) < 0 {
		t.Logf("Note: encrypted result '%s' is not in dictionary (this is OK depending on dictionary content)", encrypted)
	}
}

// TestPositionDependence verifies that same character at different positions encrypts differently
func TestPositionDependence(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	input := "aaaa"
	encrypted := sdk.ObfuscateWord(input)
	t.Logf("Input: %s -> Encrypted: %s", input, encrypted)

	// Verify that not all characters are the same (position-dependent)
	allSame := true
	if len(encrypted) > 1 {
		firstChar := encrypted[0]
		for i := 1; i < len(encrypted); i++ {
			if encrypted[i] != firstChar {
				allSame = false
				break
			}
		}
	}

	if allSame && len(encrypted) > 1 {
		t.Errorf("Position-dependence not working: all characters are the same in '%s'", encrypted)
	}

	// Verify reversibility
	decrypted := sdk.DeobfuscateWord(encrypted)
	if decrypted != input {
		t.Errorf("Reversibility failed: expected %s, got %s", input, decrypted)
	}
}

// TestDifferentSeeds verifies that different seeds produce different results
func TestDifferentSeeds(t *testing.T) {
	sdk1 := NewObfuscatorSDK(12345)
	sdk2 := NewObfuscatorSDK(54321)

	input := "testword"

	encrypted1 := sdk1.ObfuscateWord(input)
	encrypted2 := sdk2.ObfuscateWord(input)

	t.Logf("Seed 12345: %s -> %s", input, encrypted1)
	t.Logf("Seed 54321: %s -> %s", input, encrypted2)

	if encrypted1 == encrypted2 {
		t.Errorf("Different seeds produced same result: %s", encrypted1)
	}

	// Verify each can decrypt with its own seed
	if sdk1.DeobfuscateWord(encrypted1) != input {
		t.Errorf("SDK1 failed to decrypt its own result")
	}
	if sdk2.DeobfuscateWord(encrypted2) != input {
		t.Errorf("SDK2 failed to decrypt its own result")
	}
}

// TestSpecialCharactersPreserved verifies that special characters are not encrypted
func TestSpecialCharactersPreserved(t *testing.T) {
	sdk := NewObfuscatorSDK(12345)

	input := "test@example.com"
	encrypted := sdk.ObfuscateWord(input)

	t.Logf("Input: %s -> Encrypted: %s", input, encrypted)

	// Check that @ and . are preserved
	atPos := -1
	dotPos := -1
	for i, ch := range input {
		if ch == '@' {
			atPos = i
		} else if ch == '.' {
			dotPos = i
		}
	}

	if atPos >= 0 && encrypted[atPos] != '@' {
		t.Errorf("@ character was not preserved at position %d", atPos)
	}
	if dotPos >= 0 && encrypted[dotPos] != '.' {
		t.Errorf(". character was not preserved at position %d", dotPos)
	}

	// Verify reversibility
	decrypted := sdk.DeobfuscateWord(encrypted)
	if decrypted != input {
		t.Errorf("Reversibility failed: expected %s, got %s", input, decrypted)
	}
}

// TestEncryptOutOfDictSwitch tests the SetEncryptOutOfDict switch
func TestEncryptOutOfDictSwitch(t *testing.T) {
	// Test data: words that are unlikely to be in dictionary
	outOfDictWords := []string{
		"xyz123",
		"test@example.com",
		"User123_Test",
	}

	t.Run("encrypt enabled (default)", func(t *testing.T) {
		sdk := NewObfuscatorSDK(12345)

		for _, word := range outOfDictWords {
			encrypted := sdk.ObfuscateWord(word)
			t.Logf("Encrypt ON: %s -> %s", word, encrypted)

			// Should be encrypted (changed)
			if encrypted == word {
				// Special case: if word contains only special characters, it won't change
				hasAlphanumeric := false
				for _, ch := range word {
					if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
						hasAlphanumeric = true
						break
					}
				}
				if hasAlphanumeric {
					t.Errorf("Expected word to be encrypted, but it remained unchanged: %s", word)
				}
			}

			// Should be reversible
			decrypted := sdk.DeobfuscateWord(encrypted)
			if decrypted != word {
				t.Errorf("Reversibility failed: expected %s, got %s", word, decrypted)
			}
		}
	})

	t.Run("encrypt disabled", func(t *testing.T) {
		sdk := NewObfuscatorSDK(12345).SetEncryptOutOfDict(false)

		for _, word := range outOfDictWords {
			encrypted := sdk.ObfuscateWord(word)
			t.Logf("Encrypt OFF: %s -> %s", word, encrypted)

			// Should remain unchanged
			if encrypted != word {
				t.Errorf("Expected word to remain unchanged, but got: %s -> %s", word, encrypted)
			}

			// Decryption should also return the same
			decrypted := sdk.DeobfuscateWord(encrypted)
			if decrypted != word {
				t.Errorf("Expected decrypted word to match original: %s, got %s", word, decrypted)
			}
		}
	})

	t.Run("dictionary words not affected by switch", func(t *testing.T) {
		dictWord := "algorithm" // assuming this is in dictionary

		sdk1 := NewObfuscatorSDK(12345).SetEncryptOutOfDict(true)
		sdk2 := NewObfuscatorSDK(12345).SetEncryptOutOfDict(false)

		enc1 := sdk1.ObfuscateWord(dictWord)
		enc2 := sdk2.ObfuscateWord(dictWord)

		t.Logf("Dict word with encrypt ON: %s -> %s", dictWord, enc1)
		t.Logf("Dict word with encrypt OFF: %s -> %s", dictWord, enc2)

		// Both should produce the same result (dictionary mapping)
		if enc1 != enc2 {
			t.Errorf("Dictionary word mapping should not be affected by switch: %s vs %s", enc1, enc2)
		}

		// Both should be reversible
		if sdk1.DeobfuscateWord(enc1) != dictWord {
			t.Errorf("Failed to decrypt with encrypt ON")
		}
		if sdk2.DeobfuscateWord(enc2) != dictWord {
			t.Errorf("Failed to decrypt with encrypt OFF")
		}
	})

	t.Run("chained configuration", func(t *testing.T) {
		// Test that SetEncryptOutOfDict returns the SDK for chaining
		sdk := NewObfuscatorSDK(12345).
			SetEncryptOutOfDict(false).
			SetEncryptOutOfDict(true).
			SetEncryptOutOfDict(false)

		word := "xyz123"
		encrypted := sdk.ObfuscateWord(word)

		// Should be unchanged (last setting is false)
		if encrypted != word {
			t.Errorf("Expected word to remain unchanged with chained config: %s -> %s", word, encrypted)
		}
	})
}

func TestCharacterEncryption1(t *testing.T) {
	sdk := NewObfuscatorSDK(300000)

	encrypted := sdk.ObfuscateWord("PROFILE")
	t.Logf("HOME -> %s", encrypted)

	decrypted := sdk.DeobfuscateWord(encrypted)
	t.Logf("Decrypted: %s", decrypted)
}
