package tokenizer_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	tiktoken "github.com/tiktoken-go/tokenizer"

	"github.com/code-koan/llm-sdk-go/providers"
	"github.com/code-koan/llm-sdk-go/providers/tokenizer"
)

func TestDetectEncoding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		model    string
		expected tokenizer.Encoding
	}{
		{name: "gpt-4o", model: "gpt-4o", expected: tokenizer.O200kBase},
		{name: "gpt-4o-mini", model: "gpt-4o-mini", expected: tokenizer.O200kBase},
		{name: "gpt-4o-2024-08-06", model: "gpt-4o-2024-08-06", expected: tokenizer.O200kBase},
		{name: "o1", model: "o1", expected: tokenizer.O200kBase},
		{name: "o1-preview", model: "o1-preview", expected: tokenizer.O200kBase},
		{name: "o3-mini", model: "o3-mini", expected: tokenizer.O200kBase},
		{name: "o4-mini", model: "o4-mini", expected: tokenizer.O200kBase},
		{name: "chatgpt-4o-latest", model: "chatgpt-4o-latest", expected: tokenizer.O200kBase},
		{name: "gpt-4", model: "gpt-4", expected: tokenizer.Cl100kBase},
		{name: "gpt-4-turbo", model: "gpt-4-turbo", expected: tokenizer.Cl100kBase},
		{name: "gpt-3.5-turbo", model: "gpt-3.5-turbo", expected: tokenizer.Cl100kBase},
		{name: "text-embedding-3-small", model: "text-embedding-3-small", expected: tokenizer.Cl100kBase},
		{name: "claude-sonnet-4-20250514", model: "claude-sonnet-4-20250514", expected: tokenizer.Claude},
		{name: "claude-3-5-haiku-latest", model: "claude-3-5-haiku-latest", expected: tokenizer.Claude},
		{name: "gemini-2.5-flash", model: "gemini-2.5-flash", expected: tokenizer.Gemini},
		{name: "gemini-2.5-pro", model: "gemini-2.5-pro", expected: tokenizer.Gemini},
		{name: "llama-3.1-8b", model: "llama-3.1-8b", expected: tokenizer.Cl100kBase},
		{name: "deepseek-chat", model: "deepseek-chat", expected: tokenizer.Cl100kBase},
		{name: "empty string", model: "", expected: tokenizer.Cl100kBase},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tokenizer.DetectEncoding(tc.model)
			require.Equal(t, tc.expected, got)
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	t.Run("hello world", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "hello world"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("CJK mixed", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "Hello, 世界"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("email with at sign", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "test@example.com"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("math expression", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "a + b = c"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("multi line", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "line1\nline2\nline3"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("long CJK", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "你好世界，今天天气真不错！我们一起去公园散步吧。人工智能正在改变世界。"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 5)
	})

	t.Run("empty string", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: ""}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Equal(t, 6, count) // overhead only
	})

	t.Run("message with tool calls", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{
			Role:    providers.RoleAssistant,
			Content: "Let me check the weather",
			ToolCalls: []providers.ToolCall{
				{
					ID:   "call_1",
					Type: "function",
					Function: providers.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location": "Tokyo"}`,
					},
				},
			},
		}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("message with system prompt", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleSystem, Content: "You are a helpful assistant."}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("message with name", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "Hello", Name: "Alice"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 9) // overhead 9 + some heuristic tokens
	})
}

func TestCountTokens(t *testing.T) {
	t.Parallel()

	t.Run("empty messages", func(t *testing.T) {
		t.Parallel()
		count, err := tokenizer.CountTokens(nil, "gpt-4o")
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("single message gpt-4o", func(t *testing.T) {
		t.Parallel()
		text := "Hello, how are you today?"
		msg := providers.Message{Role: providers.RoleUser, Content: text}

		codec, err := tiktoken.Get(tiktoken.O200kBase)
		require.NoError(t, err)
		expectedCount, err := codec.Count(text)
		require.NoError(t, err)

		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		require.Equal(t, expectedCount+6, count)
	})

	t.Run("single message gpt-4", func(t *testing.T) {
		t.Parallel()
		text := "Hello, how are you today?"
		msg := providers.Message{Role: providers.RoleUser, Content: text}

		codec, err := tiktoken.Get(tiktoken.Cl100kBase)
		require.NoError(t, err)
		expectedCount, err := codec.Count(text)
		require.NoError(t, err)

		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4")
		require.NoError(t, err)
		require.Equal(t, expectedCount+6, count)
	})

	t.Run("single message claude", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "Hello, how are you?"}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "claude-sonnet-4-20250514")
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("single message gemini", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "Hello, how are you?"}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gemini-2.5-flash")
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("multi message conversation", func(t *testing.T) {
		t.Parallel()
		messages := []providers.Message{
			{Role: providers.RoleSystem, Content: "You are a helpful assistant."},
			{Role: providers.RoleUser, Content: "Hello!"},
			{Role: providers.RoleAssistant, Content: "Hi there! How can I help you today?"},
		}
		count, err := tokenizer.CountTokens(messages, "gpt-4o")
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})
}

func TestCountTokensWithEncoding(t *testing.T) {
	t.Parallel()

	t.Run("explicit O200kBase", func(t *testing.T) {
		t.Parallel()
		text := "The quick brown fox jumps over the lazy dog."
		msg := providers.Message{Role: providers.RoleUser, Content: text}

		codec, err := tiktoken.Get(tiktoken.O200kBase)
		require.NoError(t, err)
		expectedCount, err := codec.Count(text)
		require.NoError(t, err)

		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.O200kBase)
		require.NoError(t, err)
		require.Equal(t, expectedCount+6, count)
	})

	t.Run("explicit Cl100kBase", func(t *testing.T) {
		t.Parallel()
		text := "The quick brown fox jumps over the lazy dog."
		msg := providers.Message{Role: providers.RoleUser, Content: text}

		codec, err := tiktoken.Get(tiktoken.Cl100kBase)
		require.NoError(t, err)
		expectedCount, err := codec.Count(text)
		require.NoError(t, err)

		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Cl100kBase)
		require.NoError(t, err)
		require.Equal(t, expectedCount+6, count)
	})

	t.Run("explicit Claude", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "Hello, world!"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("explicit Gemini", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "Hello, world!"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Gemini)
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})
}

func TestMessageOverhead(t *testing.T) {
	t.Parallel()

	t.Run("no messages", func(t *testing.T) {
		t.Parallel()
		count, err := tokenizer.CountTokens(nil, "gpt-4o")
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("1 message no name", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: ""}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Equal(t, 6, count) // 3 fixed + 3 per message
	})

	t.Run("1 message with name", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: "", Name: "test"}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Equal(t, 11, count) // overhead 9 + heuristic ~2 for "test "
	})

	t.Run("2 messages no name", func(t *testing.T) {
		t.Parallel()
		msg1 := providers.Message{Role: providers.RoleUser, Content: ""}
		msg2 := providers.Message{Role: providers.RoleAssistant, Content: ""}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg1, msg2}, tokenizer.Claude)
		require.NoError(t, err)
		require.Equal(t, 9, count) // 3 fixed + 3 + 3 per message
	})
}

func TestMultimodalTokens(t *testing.T) {
	t.Parallel()

	t.Run("image low detail", func(t *testing.T) {
		t.Parallel()
		content := []providers.ContentPart{
			{Type: "text", Text: "Describe this image:"},
			{Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/img.png", Detail: "low"}},
		}
		msg := providers.Message{Role: providers.RoleUser, Content: content}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		// text tokens + 85 (low detail image) + overhead
		require.Greater(t, count, 85)
	})

	t.Run("image high detail", func(t *testing.T) {
		t.Parallel()
		content := []providers.ContentPart{
			{Type: "text", Text: "Analyze this:"},
			{Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/img.png", Detail: "high"}},
		}
		msg := providers.Message{Role: providers.RoleUser, Content: content}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		require.Greater(t, count, 765)
	})

	t.Run("image auto detail", func(t *testing.T) {
		t.Parallel()
		content := []providers.ContentPart{
			{Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/img.png", Detail: "auto"}},
		}
		msg := providers.Message{Role: providers.RoleUser, Content: content}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		require.Greater(t, count, 765)
	})

	t.Run("image with claude encoding", func(t *testing.T) {
		t.Parallel()
		content := []providers.ContentPart{
			{Type: "text", Text: "What's in this photo?"},
			{Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/photo.jpg"}},
		}
		msg := providers.Message{Role: providers.RoleUser, Content: content}
		count, err := tokenizer.CountTokensWithEncoding([]providers.Message{msg}, tokenizer.Claude)
		require.NoError(t, err)
		require.Greater(t, count, 765)
	})

	t.Run("multiple images", func(t *testing.T) {
		t.Parallel()
		content := []providers.ContentPart{
			{Type: "text", Text: "Compare these two images:"},
			{Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/a.jpg", Detail: "high"}},
			{Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/b.jpg", Detail: "low"}},
		}
		msg := providers.Message{Role: providers.RoleUser, Content: content}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		// text + 765 (high) + 85 (low) + overhead
		require.Greater(t, count, 850)
	})

	t.Run("text only with nil image", func(t *testing.T) {
		t.Parallel()
		content := []providers.ContentPart{
			{Type: "text", Text: "Hello world"},
		}
		msg := providers.Message{Role: providers.RoleUser, Content: content}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		// same as plain text message
		require.Greater(t, count, 0)
	})

	t.Run("no content parts", func(t *testing.T) {
		t.Parallel()
		msg := providers.Message{Role: providers.RoleUser, Content: []providers.ContentPart{}}
		count, err := tokenizer.CountTokens([]providers.Message{msg}, "gpt-4o")
		require.NoError(t, err)
		require.Equal(t, 6, count) // overhead only
	})
}

func TestCountText(t *testing.T) {
	t.Parallel()

	t.Run("simple text gpt-4o", func(t *testing.T) {
		t.Parallel()
		count, err := tokenizer.CountText("Hello, world!", "gpt-4o")
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})

	t.Run("empty text", func(t *testing.T) {
		t.Parallel()
		count, err := tokenizer.CountText("", "gpt-4o")
		require.NoError(t, err)
		require.Greater(t, count, 0) // overhead only
	})

	t.Run("simple text claude", func(t *testing.T) {
		t.Parallel()
		count, err := tokenizer.CountText("Hello, world!", "claude-sonnet-4-20250514")
		require.NoError(t, err)
		require.Greater(t, count, 0)
	})
}
