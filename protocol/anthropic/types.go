package anthropic

import (
	"encoding/json"

	"github.com/code-koan/llm-sdk-go/providers"
)

// Content block type constants.
const (
	BlockTypeText       = "text"
	BlockTypeThinking   = "thinking"
	BlockTypeToolUse    = "tool_use"
	BlockTypeToolResult = "tool_result"
	BlockTypeImage      = "image"
)

// Stop reason constants.
const (
	StopReasonEndTurn      = "end_turn"
	StopReasonMaxTokens    = "max_tokens"
	StopReasonStopSequence = "stop_sequence"
	StopReasonToolUse      = "tool_use"
)

// Message role constants.
const (
	RoleAssistant = "assistant"
	RoleUser      = "user"
)

// Thinking type constants.
const (
	ThinkingTypeAuto    = "auto"
	ThinkingTypeEnabled = "enabled"
)

// --- Request types ---

// MessageRequest is an Anthropic Messages API request body.
type MessageRequest struct {
	Model         string          `json:"model"`
	Messages      []Message       `json:"messages"`
	System        any             `json:"system,omitempty"` // string | []TextBlock
	MaxTokens     int             `json:"max_tokens"`
	Metadata      *Metadata       `json:"metadata,omitempty"`
	StopSequences []string        `json:"stop_sequences,omitempty"`
	Stream        bool            `json:"stream,omitempty"`
	Temperature   *float64        `json:"temperature,omitempty"`
	TopP          *float64        `json:"top_p,omitempty"`
	TopK          *int            `json:"top_k,omitempty"`
	Tools         []Tool          `json:"tools,omitempty"`
	ToolChoice    json.RawMessage `json:"tool_choice,omitempty"`
	Thinking      *ThinkingConfig `json:"thinking,omitempty"`
}

// Message is a single Anthropic input message.
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string | []ContentBlock
}

// TextBlock is a system prompt text block.
type TextBlock struct {
	Type string `json:"type"` // always "text"
	Text string `json:"text"`
}

// ContentBlock represents one element of a content array.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`   // tool_use input (JSON object)
	Content   any             `json:"content,omitempty"` // tool_result content
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   *bool           `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
	Signature string          `json:"signature,omitempty"` // thinking block signature
	Source    *ImageSource    `json:"source,omitempty"`    // image source
}

// ImageSource represents an image source in Anthropic format.
type ImageSource struct {
	Type      string `json:"type"`       // "base64" or "url"
	MediaType string `json:"media_type"` // e.g. "image/jpeg"
	Data      string `json:"data,omitempty"`
	URL       string `json:"url,omitempty"`
}

// ThinkingConfig configures extended thinking.
type ThinkingConfig struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
}

// Tool matches the Anthropic API tool format.
type Tool struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	InputSchema *InputSchema `json:"input_schema,omitempty"`
}

// InputSchema is the JSON Schema for a tool's input.
type InputSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties,omitempty"`
	Required   []string       `json:"required,omitempty"`
}

// Metadata holds per-request Anthropic metadata.
type Metadata struct {
	UserID string `json:"user_id,omitempty"`
}

// --- Response types ---

// MessageResponse is an Anthropic Messages API response body.
type MessageResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Content      []ContentBlock `json:"content"`
	Model        string         `json:"model"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        Usage          `json:"usage"`
}

// Usage is token usage in Anthropic format.
type Usage struct {
	InputTokens              int            `json:"input_tokens"`
	OutputTokens             int            `json:"output_tokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens,omitempty"`
	CacheCreation            *CacheCreation `json:"cache_creation,omitempty"`
}

// CacheCreation describes cache creation usage by TTL bucket.
type CacheCreation struct {
	Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens,omitempty"`
	Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens,omitempty"`
}

// ToSDK converts Usage to the SDK's providers.Usage format.
func (u Usage) ToSDK() *providers.Usage {
	sdkUsage := &providers.Usage{
		PromptTokens:             u.InputTokens,
		CompletionTokens:         u.OutputTokens,
		TotalTokens:              u.InputTokens + u.OutputTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
	}
	if u.CacheCreation != nil {
		sdkUsage.CacheCreation = &providers.CacheCreation{
			Ephemeral1hInputTokens: u.CacheCreation.Ephemeral1hInputTokens,
			Ephemeral5mInputTokens: u.CacheCreation.Ephemeral5mInputTokens,
		}
	}
	return sdkUsage
}

// ErrorResponse is an Anthropic-compatible error envelope.
type ErrorResponse struct {
	Type  string      `json:"type"`
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the nested error object.
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
