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
	text := extractText(messages)
	overhead := messageOverhead(messages)

	switch enc {
	case Cl100kBase, O200kBase, P50kBase, P50kEdit, R50kBase:
		codec, err := getTiktokenEncoder(enc)
		if err != nil {
			return 0, err
		}
		count, err := codec.Count(text)
		if err != nil {
			return 0, err
		}
		return count + overhead, nil
	case Claude:
		return estimateTokens(text, &claudeMultipliers) + overhead, nil
	case Gemini:
		return estimateTokens(text, &geminiMultipliers) + overhead, nil
	default:
		codec, err := getTiktokenEncoder(enc)
		if err == nil {
			count, err := codec.Count(text)
			if err == nil {
				return count + overhead, nil
			}
		}
		return estimateTokens(text, &openaiMultipliers) + overhead, nil
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

// extractText concatenates all text content from the messages into a single
// string for token counting.
func extractText(messages []providers.Message) string {
	var b strings.Builder
	for _, msg := range messages {
		if msg.Name != "" {
			b.WriteString(msg.Name)
			b.WriteString(" ")
		}
		if msg.IsMultiModal() {
			parts := msg.ContentParts()
			for _, p := range parts {
				b.WriteString(p.Text)
			}
		} else {
			b.WriteString(msg.ContentString())
		}
		for _, tc := range msg.ToolCalls {
			b.WriteString(tc.Function.Name)
			b.WriteString(" ")
			b.WriteString(tc.Function.Arguments)
		}
		if msg.Reasoning != nil {
			b.WriteString(msg.Reasoning.Content)
		}
	}
	return b.String()
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
