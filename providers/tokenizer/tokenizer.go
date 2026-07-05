package tokenizer

import (
	"errors"
	"strings"
	"sync"

	tiktoken "github.com/tiktoken-go/tokenizer"

	"github.com/code-koan/llm-sdk-go/providers"
)

// Encoding specifies the tokenizer encoding strategy.
type Encoding string

const (
	Cl100kBase Encoding = "cl100k_base"
	O200kBase  Encoding = "o200k_base"
	P50kBase   Encoding = "p50k_base"
	P50kEdit   Encoding = "p50k_edit"
	R50kBase   Encoding = "r50k_base"
	Claude     Encoding = "claude"
	Gemini     Encoding = "gemini"
)

// encoderCache caches tiktoken Codec instances by encoding name.
var encoderCache sync.Map

// DetectEncoding returns the appropriate Encoding for the given model name.
func DetectEncoding(model string) Encoding {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "gpt-4o"),
		strings.Contains(lower, "o1"),
		strings.Contains(lower, "o3"),
		strings.Contains(lower, "o4"),
		strings.Contains(lower, "chatgpt"):
		return O200kBase
	case strings.Contains(lower, "gpt-"),
		strings.Contains(lower, "text-embedding-"):
		return Cl100kBase
	case strings.Contains(lower, "claude"):
		return Claude
	case strings.Contains(lower, "gemini"):
		return Gemini
	default:
		return Cl100kBase
	}
}

// CountTokens counts the tokens for the given messages and model name.
// Returns 0 if messages is empty.
func CountTokens(messages []providers.Message, model string) (int, error) {
	if len(messages) == 0 {
		return 0, nil
	}
	enc := DetectEncoding(model)
	return CountTokensWithEncoding(messages, enc)
}

// CountTokensWithEncoding counts the tokens for the given messages using
// the specified encoding strategy.
func CountTokensWithEncoding(messages []providers.Message, enc Encoding) (int, error) {
	cc := extractContent(messages)
	overhead := messageOverhead(messages)

	switch enc {
	case Cl100kBase, O200kBase, P50kBase, P50kEdit, R50kBase:
		codec, err := getTiktokenEncoder(enc)
		if err != nil {
			return 0, err
		}
		count, err := codec.Count(cc.text)
		if err != nil {
			return 0, err
		}
		return count + cc.media + overhead, nil
	case Claude:
		return estimateTokens(cc.text, &claudeMultipliers) + cc.media + overhead, nil
	case Gemini:
		return estimateTokens(cc.text, &geminiMultipliers) + cc.media + overhead, nil
	default:
		codec, err := getTiktokenEncoder(enc)
		if err == nil {
			count, err := codec.Count(cc.text)
			if err == nil {
				return count + cc.media + overhead, nil
			}
		}
		return estimateTokens(cc.text, &openaiMultipliers) + cc.media + overhead, nil
	}
}

// CountText counts the tokens for the given text string and model name.
// It wraps the text in a user message before counting.
func CountText(text string, model string) (int, error) {
	msg := providers.Message{
		Role:    providers.RoleUser,
		Content: text,
	}
	return CountTokens([]providers.Message{msg}, model)
}

// getTiktokenEncoder returns a tiktoken Codec for the given encoding,
// using a sync.Map cache.
func getTiktokenEncoder(enc Encoding) (tiktoken.Codec, error) {
	if cached, ok := encoderCache.Load(enc); ok {
		codec, ok := cached.(tiktoken.Codec)
		if !ok {
			return nil, errors.New("cached value is not a Codec")
		}
		return codec, nil
	}
	codec, err := tiktoken.Get(tiktoken.Encoding(string(enc)))
	if err != nil {
		codec, err = tiktoken.ForModel(tiktoken.Model(string(enc)))
	}
	if err != nil {
		return nil, err
	}
	encoderCache.Store(enc, codec)
	return codec, nil
}

// contentCount holds extracted text content and media token estimates.
type contentCount struct {
	text  string
	media int
}

// extractContent extracts text content and counts media tokens from messages.
func extractContent(messages []providers.Message) contentCount {
	var b strings.Builder
	media := 0
	for _, msg := range messages {
		if msg.Name != "" {
			b.WriteString(msg.Name)
			b.WriteByte(' ')
		}
		if msg.IsMultiModal() {
			for _, part := range msg.ContentParts() {
				if part.ImageURL != nil {
					media += imageTokens(part.ImageURL)
				} else {
					// text or unknown content part type
					b.WriteString(part.Text)
				}
			}
		} else {
			b.WriteString(msg.ContentString())
		}
		for _, tc := range msg.ToolCalls {
			b.WriteString(tc.Function.Name)
			b.WriteByte(' ')
			b.WriteString(tc.Function.Arguments)
		}
		if msg.Reasoning != nil {
			b.WriteString(msg.Reasoning.Content)
		}
	}
	return contentCount{text: b.String(), media: media}
}

// imageTokens estimates token count for an image content part.
// Uses OpenAI tile-based calculation for tiktoken encodings,
// fixed conservative estimate for heuristic encodings (Claude, Gemini).
func imageTokens(img *providers.ImageURL) int {
	if img == nil {
		return 0
	}
	detail := strings.ToLower(img.Detail)
	if detail == "low" {
		return imageLowDetailTokens
	}
	// "high", "auto", or empty — assume typical 1024x1024 image.
	// After rescaling: 2048->1024, short side to 768 -> 768x768.
	// 512px tiles = ceil(768/512)^2 = 4.
	// 4 x 170 + 85 = 765.
	return imageHighDetailTokens
}

// messageOverhead calculates the fixed token overhead for the message list.
func messageOverhead(messages []providers.Message) int {
	overhead := 3
	for _, msg := range messages {
		overhead += 3
		if msg.Name != "" {
			overhead += 3
		}
	}
	return overhead
}
