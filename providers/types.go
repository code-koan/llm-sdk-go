// Package providers defines the core provider interface and related types.
package providers

import (
	"context"
	"encoding/json"
)

// Finish reasons.
const (
	FinishReasonContentFilter = "content_filter"
	FinishReasonLength        = "length"
	FinishReasonStop          = "stop"
	FinishReasonToolCalls     = "tool_calls"
)

// Cache control types.
const (
	CacheControlTypeEphemeral CacheControlType = "ephemeral"
)

// Cache control TTL values.
const (
	CacheControlTTL5m CacheControlTTL = "5m"
	CacheControlTTL1h CacheControlTTL = "1h"
)

// Reasoning effort levels for extended thinking.
const (
	ReasoningEffortAuto   ReasoningEffort = "auto"
	ReasoningEffortHigh   ReasoningEffort = "high"
	ReasoningEffortLow    ReasoningEffort = "low"
	ReasoningEffortMedium ReasoningEffort = "medium"
	ReasoningEffortNone   ReasoningEffort = "none"
)

// Message roles.
const (
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
	RoleUser      = "user"
)

// CapabilityProvider is an optional interface for providers to report capabilities.
type CapabilityProvider interface {
	Provider
	Capabilities() Capabilities
}

// EmbeddingProvider is an optional interface for providers that support embeddings.
type EmbeddingProvider interface {
	Provider
	Embedding(ctx context.Context, params EmbeddingParams) (*EmbeddingResponse, error)
}

// ErrorConverter converts provider-specific SDK errors to unified error types.
// This interface ensures all providers implement consistent error handling.
type ErrorConverter interface {
	// ConvertError takes a provider SDK error and returns a normalized error type
	// from the errors package (e.g., *errors.RateLimitError, *errors.AuthenticationError).
	ConvertError(err error) error
}

// ModelLister is an optional interface for providers that support listing models.
type ModelLister interface {
	Provider
	ListModels(ctx context.Context) (*ModelsResponse, error)
}

// Provider is the core interface that all LLM providers must implement.
type Provider interface {
	// Name returns the provider's identifier (e.g., "openai", "anthropic").
	Name() string

	// Completion performs a chat completion request.
	Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error)

	// CompletionStream performs a streaming chat completion request.
	CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error)
}

// CacheControlTTL is the provider-neutral cache control time-to-live value.
type CacheControlTTL string

// CacheControlType is the provider-neutral cache control type.
type CacheControlType string

// ProviderData holds provider-specific metadata keyed by field name.
type ProviderData map[string]any

// ReasoningEffort levels for extended thinking.
type ReasoningEffort string

// CacheControlParam specifies provider prompt caching behavior for a request element.
type CacheControlParam struct {
	TTL  CacheControlTTL  `json:"ttl,omitempty"`
	Type CacheControlType `json:"type"`
}

// CacheCreation describes cache creation usage by TTL bucket.
type CacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens,omitempty"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens,omitempty"`
}

// Capabilities describes what features a provider supports.
type Capabilities struct {
	Completion          bool
	CompletionImage     bool
	CompletionPDF       bool
	CompletionReasoning bool
	CompletionStreaming bool
	CompletionTools     bool
	Embedding           bool
	ListModels          bool
}

// ChatCompletion represents a chat completion response in OpenAI format.
type ChatCompletion struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// ChatCompletionChunk represents a streaming chunk in OpenAI format.
type ChatCompletionChunk struct {
	ID                string        `json:"id"`
	Object            string        `json:"object"`
	Created           int64         `json:"created"`
	Model             string        `json:"model"`
	Choices           []ChunkChoice `json:"choices"`
	Usage             *Usage        `json:"usage,omitempty"`
	SystemFingerprint string        `json:"system_fingerprint,omitempty"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

// ChunkChoice represents a choice in a streaming chunk.
type ChunkChoice struct {
	Index        int        `json:"index"`
	Delta        ChunkDelta `json:"delta"`
	FinishReason string     `json:"finish_reason,omitempty"`
}

// ChunkDelta represents the delta content in a streaming chunk.
type ChunkDelta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Reasoning *Reasoning `json:"reasoning,omitempty"`
}

// CompletionParams represents normalized parameters for chat completion requests.
type CompletionParams struct {
	CacheControl      *CacheControlParam `json:"cache_control,omitempty"`
	Extra             map[string]any     `json:"-"` // Extra holds per-request JSON body fields merged into the provider request body.
	Headers           map[string]string  `json:"-"`
	MaxTokens         *int               `json:"max_tokens,omitempty"`
	Messages          []Message          `json:"messages"`
	Model             string             `json:"model"`
	OverrideBody      map[string]any     `json:"-"`
	ParallelToolCalls *bool              `json:"parallel_tool_calls,omitempty"`
	ReasoningEffort   ReasoningEffort    `json:"reasoning_effort,omitempty"`
	ResponseFormat    *ResponseFormat    `json:"response_format,omitempty"`
	Seed              *int               `json:"seed,omitempty"`
	Stop              []string           `json:"stop,omitempty"`
	Stream            bool               `json:"stream,omitempty"`
	StreamOptions     *StreamOptions     `json:"stream_options,omitempty"`
	Temperature       *float64           `json:"temperature,omitempty"`
	ToolChoice        any                `json:"tool_choice,omitempty"`
	Tools             []Tool             `json:"tools,omitempty"`
	TopP              *float64           `json:"top_p,omitempty"`
	User              string             `json:"user,omitempty"`
}

// ContentPart represents a part of a multi-modal message.
type ContentPart struct {
	CacheControl *CacheControlParam `json:"cache_control,omitempty"`
	ImageURL     *ImageURL          `json:"image_url,omitempty"`
	Text         string             `json:"text,omitempty"`
	Type         string             `json:"type"`
}

// EmbeddingData represents a single embedding.
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingParams represents parameters for embedding requests.
type EmbeddingParams struct {
	Model          string `json:"model"`
	Input          any    `json:"input"`
	EncodingFormat string `json:"encoding_format,omitempty"`
	Dimensions     *int   `json:"dimensions,omitempty"`
	User           string `json:"user,omitempty"`
}

// EmbeddingResponse represents an embedding response in OpenAI format.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  *EmbeddingUsage `json:"usage,omitempty"`
}

// EmbeddingUsage represents token usage for embeddings.
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Function represents a function definition for tool calling.
type Function struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// FunctionCall represents the function being called.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ImageURL represents an image URL in a message.
type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

// JSONSchema for structured output.
type JSONSchema struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
	Strict      *bool          `json:"strict,omitempty"`
}

// Message represents a chat message in OpenAI format.
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	Name       string     `json:"name,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Reasoning  *Reasoning `json:"reasoning,omitempty"`
}

// Model represents a model from the list models API.
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents a list models response.
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// Reasoning represents extended thinking/reasoning content.
type Reasoning struct {
	Content string `json:"content,omitempty"`
}

// ResponseFormat specifies the format of the response.
type ResponseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *JSONSchema `json:"json_schema,omitempty"`
}

// StreamOptions contains options for streaming responses.
type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Tool represents a tool/function that can be called.
type Tool struct {
	CacheControl *CacheControlParam `json:"cache_control,omitempty"`
	Function     Function           `json:"function"`
	Type         string             `json:"type"`
}

// ToolCall represents a tool call made by the assistant.
type ToolCall struct {
	// Extra holds provider-specific metadata for round-tripping across
	// multi-turn conversations. Keyed by provider name (e.g. "gemini").
	// Excluded from JSON; callers preserve this through their own storage.
	Extra    map[string]ProviderData `json:"-"`
	Function FunctionCall            `json:"function"`
	ID       string                  `json:"id"`
	Type     string                  `json:"type"`
}

// ToolChoice represents a specific tool choice.
type ToolChoice struct {
	Type     string              `json:"type"`
	Function *ToolChoiceFunction `json:"function,omitempty"`
}

// ToolChoiceFunction specifies which function to call.
type ToolChoiceFunction struct {
	Name string `json:"name"`
}

// Usage represents token usage information.
type Usage struct {
	CacheCreation            *CacheCreation `json:"cache_creation,omitempty"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens,omitempty"`
	CompletionTokens         int            `json:"completion_tokens"`
	PromptTokens             int            `json:"prompt_tokens"`
	ReasoningTokens          int            `json:"reasoning_tokens,omitempty"`
	TotalTokens              int            `json:"total_tokens"`
}

// ContentParts extracts content parts from a message.
func (m *Message) ContentParts() []ContentPart {
	if m.Content == nil {
		return nil
	}

	if parts, ok := m.Content.([]ContentPart); ok {
		return parts
	}

	if parts, ok := m.Content.([]any); ok {
		result := make([]ContentPart, 0, len(parts))
		for _, p := range parts {
			if partMap, ok := p.(map[string]any); ok {
				var part ContentPart
				if b, err := json.Marshal(partMap); err == nil {
					if err := json.Unmarshal(b, &part); err == nil {
						result = append(result, part)
					}
				}
			}
		}
		return result
	}

	return nil
}

// ContentString extracts string content from a message.
func (m *Message) ContentString() string {
	if s, ok := m.Content.(string); ok {
		return s
	}
	return ""
}

// IsMultiModal returns true if the message contains multi-modal content.
func (m *Message) IsMultiModal() bool {
	return m.ContentParts() != nil
}
