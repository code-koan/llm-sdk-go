package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	Role    string         `json:"role"`
	Content any            `json:"content"` // string | []ContentBlock
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
	Input     json.RawMessage `json:"input,omitempty"`    // tool_use input (JSON object)
	Content   any             `json:"content,omitempty"`  // tool_result content
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

// --- Provider interface ---

// Provider is an optional interface for providers that natively support the
// Anthropic Messages API. Same-protocol calls bypass CompletionParams entirely,
// achieving zero-loss forward and backward conversion.
type Provider interface {
	providers.Provider
	Messages(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
	MessagesStream(ctx context.Context, req *MessageRequest) (<-chan StreamEvent, <-chan error)
}

// ToCompletionParams converts an Anthropic Messages API request to SDK
// CompletionParams. Used for the cross-protocol path when the resolved
// provider does not implement Provider.
func ToCompletionParams(req *MessageRequest) (*providers.CompletionParams, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.MaxTokens <= 0 {
		return nil, fmt.Errorf("max_tokens is required and must be greater than 0")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages is required and must not be empty")
	}

	messages := make([]providers.Message, 0, len(req.Messages)+2)

	// Extract system prompt: top-level System field first, then per-message system roles.
	systemText := extractSystemText(req.System)
	if systemText != "" {
		messages = append(messages, providers.Message{Role: providers.RoleSystem, Content: systemText})
	}

	// Convert messages.
	converted, err := convertMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	messages = append(messages, converted...)

	// Convert tools.
	tools := convertTools(req.Tools)

	// Map thinking config to reasoning effort.
	reasoningEffort := mapThinkingToEffort(req.Thinking)

	params := &providers.CompletionParams{
		Model:           req.Model,
		Messages:        messages,
		MaxTokens:       &req.MaxTokens,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		Stop:            req.StopSequences,
		Stream:          req.Stream,
		Tools:           tools,
		ReasoningEffort: reasoningEffort,
	}

	// ToolChoice: string ("auto"/"none"/"required"/"any") or object.
	if len(req.ToolChoice) > 0 {
		var choice any
		if err := json.Unmarshal(req.ToolChoice, &choice); err == nil {
			params.ToolChoice = choice
		}
	}

	// User metadata.
	if req.Metadata != nil && req.Metadata.UserID != "" {
		params.User = req.Metadata.UserID
	}

	return params, nil
}

// FromCompletion converts an SDK ChatCompletion to an Anthropic Messages API
// response. Used for the cross-protocol path.
func FromCompletion(completion *providers.ChatCompletion, req *MessageRequest) *MessageResponse {
	if completion == nil || len(completion.Choices) == 0 {
		return nil
	}

	choice := completion.Choices[0]
	model := completion.Model
	if model == "" {
		model = req.Model
	}

	// Build content blocks from the assistant message.
	var content []ContentBlock

	// Reasoning/thinking block (must come before text per Anthropic spec).
	if choice.Message.Reasoning != nil && choice.Message.Reasoning.Content != "" {
		content = append(content, ContentBlock{
			Type:      BlockTypeThinking,
			Thinking:  choice.Message.Reasoning.Content,
			Signature: choice.Message.Reasoning.Signature,
		})
	}

	// Text block.
	text := messageText(choice.Message.Content)
	if text != "" {
		content = append(content, ContentBlock{Type: BlockTypeText, Text: text})
	}

	// Tool use blocks.
	for _, tc := range choice.Message.ToolCalls {
		var input json.RawMessage
		if tc.Function.Arguments != "" {
			input = json.RawMessage(tc.Function.Arguments)
		}
		content = append(content, ContentBlock{
			Type:  BlockTypeToolUse,
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	if content == nil {
		content = []ContentBlock{}
	}

	resp := &MessageResponse{
		ID:         completion.ID,
		Type:       "message",
		Role:       RoleAssistant,
		Content:    content,
		Model:      model,
		StopReason: mapFinishReasonToStopReason(choice.FinishReason),
	}

	if completion.Usage != nil {
		resp.Usage = mapUsage(completion.Usage)
	}

	return resp
}

// --- Internal converters ---

// extractSystemText extracts a system prompt string from the Anthropic System field.
func extractSystemText(system any) string {
	switch v := system.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, item := range v {
			switch block := item.(type) {
			case map[string]any:
				if block["type"] == "text" {
					if text, ok := block["text"].(string); ok {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// convertMessages converts Anthropic messages to SDK messages.
func convertMessages(messages []Message) ([]providers.Message, error) {
	var result []providers.Message
	for i, msg := range messages {
		// System role in messages array — convert to SDK system message.
		if msg.Role == providers.RoleSystem {
			text := contentText(msg.Content)
			result = append(result, providers.Message{Role: providers.RoleSystem, Content: text})
			continue
		}

		if msg.Role != RoleUser && msg.Role != RoleAssistant {
			return nil, fmt.Errorf("messages[%d].role must be user or assistant, got %q", i, msg.Role)
		}

		switch msg.Role {
		case RoleUser:
			userMsgs, err := convertUserMessage(i, msg.Content)
			if err != nil {
				return nil, err
			}
			result = append(result, userMsgs...)
		case RoleAssistant:
			assistantMsg, err := convertAssistantMessage(i, msg.Content)
			if err != nil {
				return nil, err
			}
			result = append(result, assistantMsg)
		}
	}
	return result, nil
}

// normalizeContentBlocks converts content to a []map[string]any block list.
// Accepts string, []any, or []map[string]any.
func normalizeContentBlocks(content any) ([]map[string]any, error) {
	switch v := content.(type) {
	case string:
		return nil, nil // signal string path
	case []any:
		blocks := make([]map[string]any, 0, len(v))
		for _, item := range v {
			block, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected content block object, got %T", item)
			}
			blocks = append(blocks, block)
		}
		return blocks, nil
	case []map[string]any:
		return v, nil
	default:
		return nil, fmt.Errorf("content must be a string or an array of blocks")
	}
}

// contentText extracts a plain text string from message content.
func contentText(content any) string {
	blocks, err := normalizeContentBlocks(content)
	if err != nil || blocks == nil {
		if s, ok := content.(string); ok {
			return s
		}
		return ""
	}
	var parts []string
	for _, block := range blocks {
		if block["type"] == "text" {
			if text, ok := block["text"].(string); ok {
				parts = append(parts, text)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// convertUserMessage converts an Anthropic user message to SDK messages.
// Returns multiple messages when tool_result blocks are present.
func convertUserMessage(idx int, content any) ([]providers.Message, error) {
	blocks, err := normalizeContentBlocks(content)
	if err != nil {
		return nil, fmt.Errorf("messages[%d].content: %w", idx, err)
	}
	// String content — single user message.
	if blocks == nil {
		s, _ := content.(string)
		return []providers.Message{{Role: providers.RoleUser, Content: s}}, nil
	}

	var textParts []string
	var result []providers.Message
	flushText := func() {
		if len(textParts) > 0 {
			result = append(result, providers.Message{Role: providers.RoleUser, Content: strings.Join(textParts, "\n")})
			textParts = nil
		}
	}
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		switch blockType {
		case BlockTypeText:
			if text, ok := block["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case BlockTypeToolResult:
			flushText()
			toolUseID, _ := block["tool_use_id"].(string)
			contentStr := ""
			if c := block["content"]; c != nil {
				contentStr = fmt.Sprint(c)
			}
			result = append(result, providers.Message{
				Role:       providers.RoleTool,
				ToolCallID: toolUseID,
				Content:    contentStr,
			})
		case BlockTypeImage:
			flushText()
			src, ok := block["source"].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("messages[%d].content: image block requires source", idx)
			}
			imgURL := convertImageSource(src)
			result = append(result, providers.Message{
				Role: providers.RoleUser,
				Content: []providers.ContentPart{{
					Type:     providers.ContentTypeImageURL,
					ImageURL: imgURL,
				}},
			})
		default:
			return nil, fmt.Errorf("messages[%d].content: unsupported block type %q", idx, blockType)
		}
	}
	flushText()
	return result, nil
}

// convertImageSource converts an Anthropic image source to an SDK ImageURL.
func convertImageSource(src map[string]any) *providers.ImageURL {
	srcType, _ := src["type"].(string)
	switch srcType {
	case "url":
		if url, ok := src["url"].(string); ok {
			return &providers.ImageURL{URL: url}
		}
	case "base64":
		mediaType, _ := src["media_type"].(string)
		data, _ := src["data"].(string)
		if mediaType != "" && data != "" {
			return &providers.ImageURL{URL: "data:" + mediaType + ";base64," + data}
		}
	}
	return nil
}

// convertAssistantMessage converts an Anthropic assistant message to an SDK message.
func convertAssistantMessage(idx int, content any) (providers.Message, error) {
	blocks, err := normalizeContentBlocks(content)
	if err != nil {
		return providers.Message{}, fmt.Errorf("messages[%d].content: %w", idx, err)
	}
	// String content — simple assistant message.
	if blocks == nil {
		s, _ := content.(string)
		return providers.Message{Role: providers.RoleAssistant, Content: s}, nil
	}

	var textParts []string
	var toolCalls []providers.ToolCall
	msg := providers.Message{Role: providers.RoleAssistant}
	for _, block := range blocks {
		blockType, _ := block["type"].(string)
		switch blockType {
		case BlockTypeText:
			if text, ok := block["text"].(string); ok {
				textParts = append(textParts, text)
			}
		case BlockTypeThinking:
			thinking, _ := block["thinking"].(string)
			signature, _ := block["signature"].(string)
			msg.Reasoning = &providers.Reasoning{
				Content:   thinking,
				Signature: signature,
			}
		case BlockTypeToolUse:
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			argsStr := ""
			if input, ok := block["input"].(map[string]any); ok {
				b, _ := json.Marshal(input)
				argsStr = string(b)
			}
			toolCalls = append(toolCalls, providers.ToolCall{
				ID:   id,
				Type: "function",
				Function: providers.FunctionCall{
					Name:      name,
					Arguments: argsStr,
				},
			})
		default:
			return providers.Message{}, fmt.Errorf("messages[%d].content: unsupported block type %q", idx, blockType)
		}
	}
	if len(textParts) > 0 {
		msg.Content = strings.Join(textParts, "\n")
	}
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}
	return msg, nil
}

// convertTools converts Anthropic tools to SDK tools.
func convertTools(tools []Tool) []providers.Tool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]providers.Tool, 0, len(tools))
	for _, t := range tools {
		sdkTool := providers.Tool{
			Type: "function",
			Function: providers.Function{
				Name:        t.Name,
				Description: t.Description,
			},
		}
		if t.InputSchema != nil {
			sdkTool.Function.Parameters = make(map[string]any)
			sdkTool.Function.Parameters["type"] = t.InputSchema.Type
			if t.InputSchema.Properties != nil {
				sdkTool.Function.Parameters["properties"] = t.InputSchema.Properties
			}
			if len(t.InputSchema.Required) > 0 {
				sdkTool.Function.Parameters["required"] = t.InputSchema.Required
			}
		}
		result = append(result, sdkTool)
	}
	return result
}

// mapThinkingToEffort maps Anthropic thinking config to SDK ReasoningEffort.
func mapThinkingToEffort(thinking *ThinkingConfig) providers.ReasoningEffort {
	if thinking == nil {
		return ""
	}
	switch thinking.Type {
	case ThinkingTypeAuto:
		return providers.ReasoningEffortAuto
	case ThinkingTypeEnabled:
		switch {
		case thinking.BudgetTokens <= 1024:
			return providers.ReasoningEffortLow
		case thinking.BudgetTokens <= 4096:
			return providers.ReasoningEffortMedium
		default:
			return providers.ReasoningEffortHigh
		}
	default:
		return ""
	}
}

// mapFinishReasonToStopReason maps SDK finish reason to Anthropic stop reason.
func mapFinishReasonToStopReason(reason string) string {
	switch reason {
	case providers.FinishReasonLength:
		return StopReasonMaxTokens
	case providers.FinishReasonToolCalls:
		return StopReasonToolUse
	case providers.FinishReasonStop:
		return StopReasonEndTurn
	default:
		return StopReasonEndTurn
	}
}

// messageText extracts string content from a message's Content field.
func messageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// mapUsage converts SDK usage to Anthropic usage format.
func mapUsage(u *providers.Usage) Usage {
	usage := Usage{
		InputTokens:              u.PromptTokens,
		OutputTokens:             u.CompletionTokens,
		CacheCreationInputTokens: u.CacheCreationInputTokens,
		CacheReadInputTokens:     u.CacheReadInputTokens,
	}
	if u.CacheCreation != nil {
		usage.CacheCreation = &CacheCreation{
			Ephemeral1hInputTokens: u.CacheCreation.Ephemeral1hInputTokens,
			Ephemeral5mInputTokens: u.CacheCreation.Ephemeral5mInputTokens,
		}
	}
	return usage
}
