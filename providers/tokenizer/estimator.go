// Package tokenizer provides token estimation for LLM providers.
// Supports exact tiktoken-based counting for OpenAI models and
// character-based heuristic estimation for Claude and Gemini.
package tokenizer

import (
	"math"
	"strings"
	"unicode"
)

// Character categories for token cost multipliers.
const (
	catWord       = 0
	catNumber     = 1
	catCJK        = 2
	catSymbol     = 3
	catMathSymbol = 4
	catURLDelim   = 5
	catAtSign     = 6
	catEmoji      = 7
	catNewline    = 8
	catSpace      = 9
)

// multipliers holds per-category token cost and an additional base padding.
type multipliers struct {
	entries [10]float64
	BasePad int
}

// Provider-specific multiplier tables.
var (
	openaiMultipliers = multipliers{
		entries: [10]float64{
			1.02, // Word
			1.55, // Number
			0.85, // CJK
			0.40, // Symbol
			2.68, // MathSymbol
			1.00, // URLDelim
			2.00, // AtSign
			2.12, // Emoji
			0.50, // Newline
			0.42, // Space
		},
	}
	claudeMultipliers = multipliers{
		entries: [10]float64{
			1.13, // Word
			1.63, // Number
			1.21, // CJK
			0.40, // Symbol
			4.52, // MathSymbol
			1.26, // URLDelim
			2.82, // AtSign
			2.60, // Emoji
			0.89, // Newline
			0.39, // Space
		},
	}
	geminiMultipliers = multipliers{
		entries: [10]float64{
			1.15, // Word
			2.80, // Number
			0.68, // CJK
			0.38, // Symbol
			1.05, // MathSymbol
			1.20, // URLDelim
			2.50, // AtSign
			1.08, // Emoji
			1.15, // Newline
			0.20, // Space
		},
	}
)

// wordType tracks consecutive character type for word-level billing.
type wordType int

const (
	wordNone   wordType = 0
	wordLatin  wordType = 1
	wordNumber wordType = 2
)

// isCJK returns true if the rune is CJK (Chinese, Japanese, or Korean).
func isCJK(r rune) bool {
	if unicode.Is(unicode.Han, r) {
		return true
	}
	// Hiragana and Katakana (0x3040-0x30FF)
	if r >= 0x3040 && r <= 0x30FF {
		return true
	}
	// Hangul Syllables (0xAC00-0xD7A3)
	if r >= 0xAC00 && r <= 0xD7A3 {
		return true
	}
	return false
}

// isEmoji returns true if the rune is an emoji character.
func isEmoji(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1F9FF) ||
		(r >= 0x2600 && r <= 0x26FF) ||
		(r >= 0x2700 && r <= 0x27BF) ||
		(r >= 0x1F600 && r <= 0x1F64F) ||
		(r >= 0x1F900 && r <= 0x1F9FF) ||
		(r >= 0x1FA00 && r <= 0x1FAFF)
}

// isMathSymbol returns true if the rune is a mathematical symbol.
func isMathSymbol(r rune) bool {
	if (r >= 0x2200 && r <= 0x22FF) ||
		(r >= 0x2A00 && r <= 0x2AFF) ||
		(r >= 0x1D400 && r <= 0x1D7FF) {
		return true
	}
	return strings.ContainsRune("∑∫∂√∞≤≥≠≈±×÷∈∉∋∌⊂⊃⊆⊇∪∩∧∨¬∀∃∄∅∆∇∝∟∠∡∢°′″‴⁺⁻⁼⁽⁾ⁿ₀₁₂₃₄₅₆₇₈₉₊₋₌₍₎²³¹⁴⁵⁶⁷⁸⁹⁰", r)
}

// isURLDelim returns true if the rune is a URL delimiter.
func isURLDelim(r rune) bool {
	return strings.ContainsRune("/:?&=;#%", r)
}

// isLatinOrNumber returns true if the rune is a letter or a number.
func isLatinOrNumber(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

// estimateTokens estimates the token count for the given text using the
// provided multiplier table. Returns the estimated token count including
// base padding.
func estimateTokens(text string, mult *multipliers) int {
	var currentWordType wordType
	var total float64

	for _, r := range text {
		if unicode.IsSpace(r) {
			currentWordType = wordNone
			if r == '\n' || r == '\t' {
				total += mult.entries[catNewline]
			} else {
				total += mult.entries[catSpace]
			}
			continue
		}

		if isCJK(r) {
			currentWordType = wordNone
			total += mult.entries[catCJK]
			continue
		}

		if isEmoji(r) {
			currentWordType = wordNone
			total += mult.entries[catEmoji]
			continue
		}

		if isLatinOrNumber(r) {
			var newType wordType
			if unicode.IsNumber(r) {
				newType = wordNumber
			} else {
				newType = wordLatin
			}
			if currentWordType == wordNone || currentWordType != newType {
				if newType == wordLatin {
					total += mult.entries[catWord]
				} else {
					total += mult.entries[catNumber]
				}
			}
			currentWordType = newType
			continue
		}

		// Symbols
		currentWordType = wordNone
		switch {
		case isMathSymbol(r):
			total += mult.entries[catMathSymbol]
		case r == '@':
			total += mult.entries[catAtSign]
		case isURLDelim(r):
			total += mult.entries[catURLDelim]
		default:
			total += mult.entries[catSymbol]
		}
	}

	return int(math.Ceil(total)) + mult.BasePad
}
