package openai

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/config"
	"github.com/code-koan/llm-sdk-go/errors"
	"github.com/code-koan/llm-sdk-go/providers"
)

func TestNewCompatible(t *testing.T) {
	// Note: Not using t.Parallel() here because child test uses t.Setenv.

	t.Run("creates provider with valid config", func(t *testing.T) {
		t.Parallel()

		baseCfg := CompatibleConfig{
			Name:           "test-provider",
			DefaultBaseURL: "http://localhost:8080/v1",
			DefaultAPIKey:  "test-key",
			RequireAPIKey:  false,
			Capabilities: providers.Capabilities{
				Completion: true,
			},
		}

		provider, err := NewCompatible(baseCfg)
		require.NoError(t, err)
		require.NotNil(t, provider)
		require.Equal(t, "test-provider", provider.Name())
	})

	t.Run("returns error when name is missing", func(t *testing.T) {
		t.Parallel()

		baseCfg := CompatibleConfig{
			DefaultBaseURL: "http://localhost:8080/v1",
		}

		provider, err := NewCompatible(baseCfg)
		require.Error(t, err)
		require.Nil(t, provider)
		require.Contains(t, err.Error(), "provider name is required")
	})

	t.Run("returns error when API key required but missing", func(t *testing.T) {
		t.Parallel()

		baseCfg := CompatibleConfig{
			Name:          "test-provider",
			APIKeyEnvVar:  "TEST_API_KEY",
			RequireAPIKey: true,
		}

		provider, err := NewCompatible(baseCfg)
		require.Error(t, err)
		require.Nil(t, provider)

		var missingKeyErr *errors.MissingAPIKeyError
		require.ErrorAs(t, err, &missingKeyErr)
	})

	t.Run("uses default API key when not required", func(t *testing.T) {
		t.Parallel()

		baseCfg := CompatibleConfig{
			Name:          "test-provider",
			DefaultAPIKey: "default-key",
			RequireAPIKey: false,
		}

		provider, err := NewCompatible(baseCfg)
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("uses config base URL over default", func(t *testing.T) {
		t.Parallel()

		baseCfg := CompatibleConfig{
			Name:           "test-provider",
			DefaultBaseURL: "http://default:8080/v1",
			DefaultAPIKey:  "test-key",
		}

		provider, err := NewCompatible(baseCfg, config.WithBaseURL("http://custom:9090/v1"))
		require.NoError(t, err)
		require.NotNil(t, provider)
	})

	t.Run("uses environment variable for base URL", func(t *testing.T) {
		t.Setenv("TEST_BASE_URL", "http://env:8080/v1")

		baseCfg := CompatibleConfig{
			Name:           "test-provider",
			BaseURLEnvVar:  "TEST_BASE_URL",
			DefaultBaseURL: "http://default:8080/v1",
			DefaultAPIKey:  "test-key",
		}

		provider, err := NewCompatible(baseCfg)
		require.NoError(t, err)
		require.NotNil(t, provider)
	})
}

func TestCompatibleProviderCapabilities(t *testing.T) {
	t.Parallel()

	expectedCaps := providers.Capabilities{
		Completion:          true,
		CompletionStreaming: true,
		Embedding:           true,
	}

	baseCfg := CompatibleConfig{
		Name:         "test-provider",
		Capabilities: expectedCaps,
	}

	provider, err := NewCompatible(baseCfg)
	require.NoError(t, err)

	caps := provider.Capabilities()
	require.Equal(t, expectedCaps, caps)
}

func TestValidateCompletionParams(t *testing.T) {
	t.Parallel()

	t.Run("returns error when model is empty", func(t *testing.T) {
		t.Parallel()

		params := providers.CompletionParams{
			Messages: []providers.Message{{Role: providers.RoleUser, Content: "Hello"}},
		}

		err := validateCompletionParams(params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "model is required")
	})

	t.Run("returns error when messages is empty", func(t *testing.T) {
		t.Parallel()

		params := providers.CompletionParams{
			Model:    "gpt-4",
			Messages: []providers.Message{},
		}

		err := validateCompletionParams(params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "at least one message is required")
	})

	t.Run("returns error for unknown message role", func(t *testing.T) {
		t.Parallel()

		params := providers.CompletionParams{
			Model: "gpt-4",
			Messages: []providers.Message{
				{Role: "unknown_role", Content: "Hello"},
			},
		}

		err := validateCompletionParams(params)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown message role")
	})

	t.Run("accepts valid params", func(t *testing.T) {
		t.Parallel()

		params := providers.CompletionParams{
			Model: "gpt-4",
			Messages: []providers.Message{
				{Role: providers.RoleUser, Content: "Hello"},
			},
		}

		err := validateCompletionParams(params)
		require.NoError(t, err)
	})
}

func TestConvertResponseFormat(t *testing.T) {
	t.Parallel()

	t.Run("handles nil format", func(t *testing.T) {
		t.Parallel()

		result := convertResponseFormat(nil)
		require.NotNil(t, result)
	})

	t.Run("converts json_object format", func(t *testing.T) {
		t.Parallel()

		format := &providers.ResponseFormat{Type: responseFormatJSONObject}
		result := convertResponseFormat(format)
		require.NotNil(t, result.OfJSONObject)
	})

	t.Run("converts json_schema format", func(t *testing.T) {
		t.Parallel()

		strict := true
		format := &providers.ResponseFormat{
			Type: responseFormatJSONSchema,
			JSONSchema: &providers.JSONSchema{
				Name:        "test_schema",
				Description: "Test schema",
				Schema:      map[string]any{"type": "object"},
				Strict:      &strict,
			},
		}
		result := convertResponseFormat(format)
		require.NotNil(t, result.OfJSONSchema)
	})

	t.Run("defaults to text format for unknown type", func(t *testing.T) {
		t.Parallel()

		format := &providers.ResponseFormat{Type: "unknown"}
		result := convertResponseFormat(format)
		require.NotNil(t, result.OfText)
	})
}

func TestConvertEmbeddingParams(t *testing.T) {
	t.Parallel()

	t.Run("converts string input", func(t *testing.T) {
		t.Parallel()

		params := providers.EmbeddingParams{
			Model: "text-embedding-3-small",
			Input: "Hello, world!",
		}

		result := convertEmbeddingParams(params, "")
		require.NotNil(t, result.Input.OfString)
	})

	t.Run("converts string array input", func(t *testing.T) {
		t.Parallel()

		params := providers.EmbeddingParams{
			Model: "text-embedding-3-small",
			Input: []string{"Hello", "World"},
		}

		result := convertEmbeddingParams(params, "")
		require.NotNil(t, result.Input.OfArrayOfStrings)
	})

	t.Run("handles unknown input type", func(t *testing.T) {
		t.Parallel()

		params := providers.EmbeddingParams{
			Model: "text-embedding-3-small",
			Input: 12345, // Unsupported type.
		}

		result := convertEmbeddingParams(params, "")
		// Should convert to string representation.
		require.NotNil(t, result.Input.OfString)
	})

	t.Run("includes optional parameters", func(t *testing.T) {
		t.Parallel()

		dims := 256
		params := providers.EmbeddingParams{
			Model:          "text-embedding-3-small",
			Input:          "Hello",
			EncodingFormat: "float",
			Dimensions:     &dims,
			User:           "test-user",
		}

		result := convertEmbeddingParams(params, "")
		require.Equal(t, int64(256), result.Dimensions.Value)
		require.Equal(t, "test-user", result.User.Value)
	})
}

func TestCompatibleHeaders(t *testing.T) {
	t.Parallel()

	// Fake server that captures request headers.
	var capturedHeaders map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = map[string]string{
			"X-Custom-Header": r.Header.Get("X-Custom-Header"),
			"Authorization":   r.Header.Get("Authorization"),
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1700000000,
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "hello"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
		}`))
	}))
	t.Cleanup(srv.Close)

	baseCfg := CompatibleConfig{
		Name:           "test-provider",
		DefaultBaseURL: srv.URL,
		DefaultAPIKey:  "test-key",
		Capabilities: providers.Capabilities{
			Completion: true,
		},
	}

	provider, err := NewCompatible(baseCfg)
	require.NoError(t, err)

	params := providers.CompletionParams{
		Model:    "test-model",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "Hello"}},
		Headers:  map[string]string{"X-Custom-Header": "custom-value"},
	}

	_, err = provider.Completion(context.Background(), params)
	require.NoError(t, err)
	require.NotNil(t, capturedHeaders)
	require.Equal(t, "custom-value", capturedHeaders["X-Custom-Header"])
}

func TestCompatibleExtraConflict(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		extra map[string]any
	}{
		{
			name:  "model conflict",
			extra: map[string]any{"model": "gpt-5"},
		},
		{
			name:  "temperature conflict",
			extra: map[string]any{"temperature": 0.5},
		},
		{
			name:  "user conflict",
			extra: map[string]any{"user": "custom-user"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baseCfg := CompatibleConfig{
				Name:           "test-provider",
				DefaultBaseURL: "http://localhost:9999",
				DefaultAPIKey:  "test-key",
				Capabilities: providers.Capabilities{
					Completion: true,
				},
			}

			provider, err := NewCompatible(baseCfg)
			require.NoError(t, err)

			params := providers.CompletionParams{
				Model:    "test-model",
				Messages: []providers.Message{{Role: providers.RoleUser, Content: "Hello"}},
				Extra:    tc.extra,
			}

			_, err = provider.Completion(context.Background(), params)
			require.Error(t, err)

			var unsupportedErr *errors.UnsupportedParamError
			require.ErrorAs(t, err, &unsupportedErr)
		})
	}
}

func TestCompatibleOverrideBody(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &capturedBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-test",
			"object": "chat.completion",
			"created": 1700000000,
			"model": "test-model",
			"choices": [{
				"index": 0,
				"message": {"role": "assistant", "content": "hello"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
		}`))
	}))
	t.Cleanup(srv.Close)

	baseCfg := CompatibleConfig{
		Name:           "test-provider",
		DefaultBaseURL: srv.URL,
		DefaultAPIKey:  "test-key",
		Capabilities: providers.Capabilities{
			Completion: true,
		},
	}

	provider, err := NewCompatible(baseCfg)
	require.NoError(t, err)

	params := providers.CompletionParams{
		Model:    "test-model",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "Hello"}},
		OverrideBody: map[string]any{
			"model": "overridden-model",
		},
	}

	_, err = provider.Completion(context.Background(), params)
	require.NoError(t, err)
	require.Equal(t, "overridden-model", capturedBody["model"])
}

func TestCompatibleDefaultUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		paramsUser  string
		defaultUser string
		wantUser    any
	}{
		{
			name:        "params user takes precedence",
			paramsUser:  "params-user",
			defaultUser: "default-user",
			wantUser:    "params-user",
		},
		{
			name:        "default user used when params user empty",
			paramsUser:  "",
			defaultUser: "default-user",
			wantUser:    "default-user",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var capturedBody map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &capturedBody)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"id": "chatcmpl-test",
					"object": "chat.completion",
					"created": 1700000000,
					"model": "test-model",
					"choices": [{
						"index": 0,
						"message": {"role": "assistant", "content": "hello"},
						"finish_reason": "stop"
					}],
					"usage": {"prompt_tokens": 5, "completion_tokens": 3, "total_tokens": 8}
				}`))
			}))
			t.Cleanup(srv.Close)

			opts := []config.Option{
				config.WithAPIKey("test-key"),
				config.WithBaseURL(srv.URL),
			}
			if tc.defaultUser != "" {
				opts = append(opts, config.WithUserID(tc.defaultUser))
			}

			baseCfg := CompatibleConfig{
				Name: "test-provider",
				Capabilities: providers.Capabilities{
					Completion: true,
				},
			}

			provider, err := NewCompatible(baseCfg, opts...)
			require.NoError(t, err)

			params := providers.CompletionParams{
				Model:    "test-model",
				Messages: []providers.Message{{Role: providers.RoleUser, Content: "Hello"}},
				User:     tc.paramsUser,
			}

			_, err = provider.Completion(context.Background(), params)
			require.NoError(t, err)
			require.Equal(t, tc.wantUser, capturedBody["user"])
		})
	}
}

func TestConvertResponseCachedTokens(t *testing.T) {
	t.Parallel()

	t.Run("maps cached_tokens to CacheReadInputTokens", func(t *testing.T) {
		t.Parallel()

		resp := &openai.ChatCompletion{
			ID:      "test-id",
			Object:  objectChatCompletion,
			Created: 1700000000,
			Model:   "test-model",
			Choices: []openai.ChatCompletionChoice{
				{
					Index:        0,
					FinishReason: "stop",
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "hello",
					},
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				PromptTokensDetails: openai.CompletionUsagePromptTokensDetails{
					CachedTokens: 3,
				},
			},
		}

		result := convertResponse(resp)
		require.NotNil(t, result.Usage)
		require.Equal(t, 10, result.Usage.PromptTokens)
		require.Equal(t, 5, result.Usage.CompletionTokens)
		require.Equal(t, 15, result.Usage.TotalTokens)
		require.Equal(t, 3, result.Usage.CacheReadInputTokens)
	})

	t.Run("preserves reasoning tokens mapping", func(t *testing.T) {
		t.Parallel()

		resp := &openai.ChatCompletion{
			ID:      "test-id",
			Object:  objectChatCompletion,
			Created: 1700000000,
			Model:   "test-model",
			Choices: []openai.ChatCompletionChoice{
				{
					Index:        0,
					FinishReason: "stop",
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "hello",
					},
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				CompletionTokensDetails: openai.CompletionUsageCompletionTokensDetails{
					ReasoningTokens: 2,
				},
			},
		}

		result := convertResponse(resp)
		require.NotNil(t, result.Usage)
		require.Equal(t, 2, result.Usage.ReasoningTokens)
	})

	t.Run("no usage when all zero", func(t *testing.T) {
		t.Parallel()

		resp := &openai.ChatCompletion{
			ID:      "test-id",
			Object:  objectChatCompletion,
			Created: 1700000000,
			Model:   "test-model",
			Choices: []openai.ChatCompletionChoice{
				{
					Index:        0,
					FinishReason: "stop",
					Message: openai.ChatCompletionMessage{
						Role:    "assistant",
						Content: "hello",
					},
				},
			},
			Usage: openai.CompletionUsage{},
		}

		result := convertResponse(resp)
		require.Nil(t, result.Usage)
	})
}

func TestConvertChunkCachedTokens(t *testing.T) {
	t.Parallel()

	t.Run("maps cached_tokens in chunk", func(t *testing.T) {
		t.Parallel()

		chunk := &openai.ChatCompletionChunk{
			ID:      "test-chunk",
			Created: 1700000000,
			Model:   "test-model",
			Choices: []openai.ChatCompletionChunkChoice{
				{
					Index: 0,
					Delta: openai.ChatCompletionChunkChoiceDelta{
						Role:    "assistant",
						Content: "hello",
					},
				},
			},
			Usage: openai.CompletionUsage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
				PromptTokensDetails: openai.CompletionUsagePromptTokensDetails{
					CachedTokens: 3,
				},
			},
		}

		result := convertChunk(chunk)
		require.NotNil(t, result.Usage)
		require.Equal(t, 10, result.Usage.PromptTokens)
		require.Equal(t, 5, result.Usage.CompletionTokens)
		require.Equal(t, 15, result.Usage.TotalTokens)
		require.Equal(t, 3, result.Usage.CacheReadInputTokens)
	})

	t.Run("no usage when all zero", func(t *testing.T) {
		t.Parallel()

		chunk := &openai.ChatCompletionChunk{
			ID:      "test-chunk",
			Created: 1700000000,
			Model:   "test-model",
			Choices: []openai.ChatCompletionChunkChoice{},
			Usage:   openai.CompletionUsage{},
		}

		result := convertChunk(chunk)
		require.Nil(t, result.Usage)
	})
}

func TestConvertAPIErrorRateLimitWithHeaders(t *testing.T) {
	t.Parallel()

	t.Run("populates rate limit headers from response", func(t *testing.T) {
		t.Parallel()

		err := convertAPIError("test-provider", &openai.Error{
			StatusCode: 429,
			Code:       apiCodeRateLimitExceeded,
			Message:    "rate limited",
			Response: &http.Response{
				StatusCode: 429,
				Header: http.Header{
					"Retry-After":                  {"30"},
					"X-RateLimit-Remaining-Tokens": {"100"},
				},
			},
		}, stderrors.New("rate limit exceeded"))

		var rateErr *errors.RateLimitError
		require.ErrorAs(t, err, &rateErr)
		require.Equal(t, "test-provider", rateErr.Provider)
		require.NotNil(t, rateErr.Headers)
		require.Equal(t, "30", rateErr.Headers["Retry-After"])
		require.Equal(t, "30", rateErr.Headers["Retry-After"])
		require.Equal(t, 30, rateErr.RetryAfter)
	})

	t.Run("falls back to basic rate limit error when response is nil", func(t *testing.T) {
		t.Parallel()

		err := convertAPIError("test-provider", &openai.Error{
			StatusCode: 429,
			Code:       apiCodeRateLimitExceeded,
			Message:    "rate limited",
			Response:   nil,
		}, stderrors.New("rate limit exceeded"))

		var rateErr *errors.RateLimitError
		require.ErrorAs(t, err, &rateErr)
		require.Equal(t, "test-provider", rateErr.Provider)
		require.Nil(t, rateErr.Headers)
		require.Equal(t, 0, rateErr.RetryAfter)
	})
}

func TestStreamingContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("respects context cancellation", func(t *testing.T) {
		t.Parallel()

		baseCfg := CompatibleConfig{
			Name:           "test-provider",
			DefaultBaseURL: "http://localhost:9999/v1", // Non-existent server.
			DefaultAPIKey:  "test-key",
		}

		provider, err := NewCompatible(baseCfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately.

		params := providers.CompletionParams{
			Model:    "test-model",
			Messages: []providers.Message{{Role: providers.RoleUser, Content: "Hello"}},
		}

		chunks, errs := provider.CompletionStream(ctx, params)

		// Drain channels.
		for range chunks {
		}
		<-errs

		// Test passes if it doesn't hang.
	})
}
