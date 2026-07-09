package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/providers"
)

func TestToCompletionParams_SimpleUserMessage(t *testing.T) {
	t.Parallel()

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "user", Content: "Hello, world!"},
		},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Equal(t, "claude-sonnet-4-20250514", params.Model)
	require.Equal(t, 4096, *params.MaxTokens)
	require.Len(t, params.Messages, 1)
	require.Equal(t, providers.RoleUser, params.Messages[0].Role)
	require.Equal(t, "Hello, world!", params.Messages[0].ContentString())
}

func TestToCompletionParams_WithSystem(t *testing.T) {
	t.Parallel()

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		System:    "You are helpful.",
		Messages: []Message{
			{Role: "user", Content: "Hi"},
		},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Messages, 2)
	require.Equal(t, providers.RoleSystem, params.Messages[0].Role)
	require.Equal(t, "You are helpful.", params.Messages[0].ContentString())
	require.Equal(t, providers.RoleUser, params.Messages[1].Role)
}

func TestToCompletionParams_Validation(t *testing.T) {
	t.Parallel()

	t.Run("empty model", func(t *testing.T) {
		t.Parallel()
		_, err := ToCompletionParams(&MessageRequest{Model: "", MaxTokens: 100, Messages: []Message{{Role: "user", Content: "hi"}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "model")
	})

	t.Run("zero max_tokens", func(t *testing.T) {
		t.Parallel()
		_, err := ToCompletionParams(&MessageRequest{Model: "claude", Messages: []Message{{Role: "user", Content: "hi"}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "max_tokens")
	})

	t.Run("empty messages", func(t *testing.T) {
		t.Parallel()
		_, err := ToCompletionParams(&MessageRequest{Model: "claude", MaxTokens: 100})
		require.Error(t, err)
		require.Contains(t, err.Error(), "messages")
	})
}

func TestToCompletionParams_WithTools(t *testing.T) {
	t.Parallel()

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "user", Content: "What is the weather?"},
		},
		Tools: []Tool{
			{
				Name:        "get_weather",
				Description: "Get weather for a location",
				InputSchema: &InputSchema{
					Type: "object",
					Properties: map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": "City name",
						},
					},
					Required: []string{"location"},
				},
			},
		},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Tools, 1)
	require.Equal(t, "function", params.Tools[0].Type)
	require.Equal(t, "get_weather", params.Tools[0].Function.Name)
	require.Equal(t, "Get weather for a location", params.Tools[0].Function.Description)
}

func TestToCompletionParams_WithThinking(t *testing.T) {
	t.Parallel()

	t.Run("auto", func(t *testing.T) {
		t.Parallel()
		req := &MessageRequest{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			Messages:  []Message{{Role: "user", Content: "Think deeply"}},
			Thinking:  &ThinkingConfig{Type: "auto"},
		}
		params, err := ToCompletionParams(req)
		require.NoError(t, err)
		require.Equal(t, providers.ReasoningEffortAuto, params.ReasoningEffort)
	})

	t.Run("enabled with budget", func(t *testing.T) {
		t.Parallel()
		req := &MessageRequest{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			Messages:  []Message{{Role: "user", Content: "Think deeply"}},
			Thinking:  &ThinkingConfig{Type: "enabled", BudgetTokens: 16000},
		}
		params, err := ToCompletionParams(req)
		require.NoError(t, err)
		require.Equal(t, providers.ReasoningEffortHigh, params.ReasoningEffort)
	})
}

func TestToCompletionParams_WithToolResult(t *testing.T) {
	t.Parallel()

	content := []any{
		map[string]any{"type": "tool_result", "tool_use_id": "tool_123", "content": "result data"},
	}

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "user", Content: content},
		},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Messages, 1)
	require.Equal(t, providers.RoleTool, params.Messages[0].Role)
	require.Equal(t, "tool_123", params.Messages[0].ToolCallID)
}

func TestToCompletionParams_MessageWithToolUse(t *testing.T) {
	t.Parallel()

	content := []map[string]any{
		{"type": "text", "text": "Let me check the weather."},
		{"type": "tool_use", "id": "call_001", "name": "get_weather", "input": map[string]any{"location": "Paris"}},
	}

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "user", Content: "What's the weather?"},
			{Role: "assistant", Content: content},
		},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Messages, 2) // user + assistant

	// Check that the assistant message contains both text and tool calls.
	assistantMsg := params.Messages[1]
	require.Equal(t, providers.RoleAssistant, assistantMsg.Role)
	toolCalls := assistantMsg.ToolCalls
	require.NotEmpty(t, toolCalls)

	// Find the tool_use call.
	hasToolCall := false
	for _, tc := range toolCalls {
		if tc.Function.Name == "get_weather" {
			hasToolCall = true
			require.Equal(t, "call_001", tc.ID)
			require.Contains(t, tc.Function.Arguments, "Paris")
		}
	}
	require.True(t, hasToolCall)
}

func TestToCompletionParams_WithThinkingBlock(t *testing.T) {
	t.Parallel()

	content := []map[string]any{
		{"type": "thinking", "thinking": "Let me reason about this...", "signature": "sig_abc123"},
		{"type": "text", "text": "Based on my reasoning, the answer is 42."},
	}

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		Messages: []Message{
			{Role: "assistant", Content: content},
		},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Messages, 1)
	require.NotNil(t, params.Messages[0].Reasoning)
	require.Equal(t, "Let me reason about this...", params.Messages[0].Reasoning.Content)
	require.Equal(t, "sig_abc123", params.Messages[0].Reasoning.Signature)
}

func TestFromCompletion_SimpleText(t *testing.T) {
	t.Parallel()

	completion := &providers.ChatCompletion{
		ID:    "msg_001",
		Model: "claude-sonnet-4-20250514",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "Hello!",
			},
			FinishReason: providers.FinishReasonStop,
		}},
		Usage: &providers.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}

	resp := FromCompletion(completion, &MessageRequest{Model: "claude-sonnet-4-20250514"})
	require.NotNil(t, resp)
	require.Equal(t, "msg_001", resp.ID)
	require.Equal(t, "message", resp.Type)
	require.Equal(t, "assistant", resp.Role)
	require.Len(t, resp.Content, 1)
	require.Equal(t, BlockTypeText, resp.Content[0].Type)
	require.Equal(t, "Hello!", resp.Content[0].Text)
	require.Equal(t, StopReasonEndTurn, resp.StopReason)
	require.Equal(t, 10, resp.Usage.InputTokens)
	require.Equal(t, 20, resp.Usage.OutputTokens)
}

func TestFromCompletion_WithReasoning(t *testing.T) {
	t.Parallel()

	completion := &providers.ChatCompletion{
		ID:    "msg_002",
		Model: "claude-sonnet-4-20250514",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "The answer is 42.",
				Reasoning: &providers.Reasoning{
					Content:   "I need to compute the meaning of life...",
					Signature: "sig_xyz",
				},
			},
			FinishReason: providers.FinishReasonStop,
		}},
	}

	resp := FromCompletion(completion, &MessageRequest{Model: "claude-sonnet-4-20250514"})
	require.NotNil(t, resp)
	// Should have thinking block first, then text block.
	require.Len(t, resp.Content, 2)
	require.Equal(t, BlockTypeThinking, resp.Content[0].Type)
	require.Equal(t, "I need to compute the meaning of life...", resp.Content[0].Thinking)
	require.Equal(t, "sig_xyz", resp.Content[0].Signature)
	require.Equal(t, BlockTypeText, resp.Content[1].Type)
	require.Equal(t, "The answer is 42.", resp.Content[1].Text)
}

func TestFromCompletion_WithToolCalls(t *testing.T) {
	t.Parallel()

	completion := &providers.ChatCompletion{
		ID:    "msg_003",
		Model: "claude-sonnet-4-20250514",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "Calling the weather tool.",
				ToolCalls: []providers.ToolCall{{
					ID:   "call_001",
					Type: "function",
					Function: providers.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"Paris"}`,
					},
				}},
			},
			FinishReason: providers.FinishReasonToolCalls,
		}},
	}

	resp := FromCompletion(completion, &MessageRequest{Model: "claude-sonnet-4-20250514"})
	require.NotNil(t, resp)
	require.Equal(t, StopReasonToolUse, resp.StopReason)
	// Should have text block + tool_use block.
	var toolUseBlock *ContentBlock
	for i := range resp.Content {
		if resp.Content[i].Type == BlockTypeToolUse {
			toolUseBlock = &resp.Content[i]
			break
		}
	}
	require.NotNil(t, toolUseBlock)
	require.Equal(t, "call_001", toolUseBlock.ID)
	require.Equal(t, "get_weather", toolUseBlock.Name)
	require.JSONEq(t, `{"location":"Paris"}`, string(toolUseBlock.Input))
}

func TestFromCompletion_NilInput(t *testing.T) {
	t.Parallel()

	resp := FromCompletion(nil, &MessageRequest{Model: "claude"})
	require.Nil(t, resp)
}

func TestStreamAdapter_TextOnly(t *testing.T) {
	t.Parallel()

	adapter := NewStreamAdapter()
	var allEvents []StreamEvent

	// Simulate chunks.
	chunks := []providers.ChatCompletionChunk{
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{Role: "assistant"}}}},
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{Content: "Hello"}}}},
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{Content: " world"}}}},
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{FinishReason: "stop"}}, Usage: &providers.Usage{CompletionTokens: 5}},
	}

	for _, chunk := range chunks {
		allEvents = append(allEvents, adapter.Adapt(chunk)...)
	}
	allEvents = append(allEvents, adapter.Flush()...)

	// Verify event sequence: message_start -> content_block_start -> content_block_delta(s) -> content_block_stop -> message_delta -> message_stop
	eventTypes := make([]string, 0, len(allEvents))
	for _, e := range allEvents {
		eventTypes = append(eventTypes, e.Type)
	}

	require.Equal(t, EventMessageStart, eventTypes[0])
	require.Contains(t, eventTypes, EventContentBlockStart)
	require.Contains(t, eventTypes, EventContentBlockDelta)
	require.Contains(t, eventTypes, EventContentBlockStop)
	require.Contains(t, eventTypes, EventMessageDelta)
	require.Equal(t, EventMessageStop, eventTypes[len(eventTypes)-1])
}

func TestStreamAdapter_WithToolCalls(t *testing.T) {
	t.Parallel()

	adapter := NewStreamAdapter()
	var allEvents []StreamEvent

	chunks := []providers.ChatCompletionChunk{
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{Role: "assistant"}}}},
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{
			ToolCalls: []providers.ToolCall{{
				ID:   "call_001",
				Type: "function",
				Function: providers.FunctionCall{Name: "get_weather"},
			}},
		}}}},
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{
			ToolCalls: []providers.ToolCall{{
				Function: providers.FunctionCall{Arguments: `{"location":"Paris"}`},
			}},
		}}}},
		{ID: "msg_001", Model: "claude", Choices: []providers.ChunkChoice{{FinishReason: "tool_calls"}}},
	}

	for _, chunk := range chunks {
		allEvents = append(allEvents, adapter.Adapt(chunk)...)
	}
	allEvents = append(allEvents, adapter.Flush()...)

	// Verify content_block_start for tool_use.
	var toolStartFound bool
	for _, e := range allEvents {
		if e.Type == EventContentBlockStart && e.ContentBlock != nil && e.ContentBlock.Type == BlockTypeToolUse {
			toolStartFound = true
			require.Equal(t, "call_001", e.ContentBlock.ID)
			require.Equal(t, "get_weather", e.ContentBlock.Name)
		}
		if e.Type == EventContentBlockDelta && e.Delta != nil {
			if d, ok := e.Delta.(InputJSONDelta); ok {
				require.Equal(t, DeltaTypeInputJSON, d.Type)
			}
		}
	}
	require.True(t, toolStartFound)
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	// Verify: Anthropic request -> CompletionParams -> ChatCompletion -> Anthropic response
	// does not lose critical fields.

	req := &MessageRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 4096,
		System:    "You are a helpful assistant.",
		Messages: []Message{
			{
				Role: "user",
				Content: []map[string]any{
					{"type": "text", "text": "What is the weather in Paris?"},
				},
			},
			{
				Role: "assistant",
				Content: []map[string]any{
					{"type": "thinking", "thinking": "I should call the weather tool.", "signature": "sig_test"},
					{"type": "tool_use", "id": "call_001", "name": "get_weather", "input": map[string]any{"location": "Paris"}},
				},
			},
			{
				Role: "user",
				Content: []map[string]any{
					{"type": "tool_result", "tool_use_id": "call_001", "content": "22°C and sunny"},
				},
			},
		},
		Thinking: &ThinkingConfig{Type: "enabled", BudgetTokens: 16000},
		Tools: []Tool{{
			Name:        "get_weather",
			Description: "Get weather for a location",
			InputSchema: &InputSchema{
				Type:       "object",
				Properties: map[string]any{"location": map[string]any{"type": "string"}},
				Required:   []string{"location"},
			},
		}},
	}

	// 1. Request -> CompletionParams
	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.NotNil(t, params)
	require.Equal(t, providers.ReasoningEffortHigh, params.ReasoningEffort)
	require.Len(t, params.Tools, 1)

	// Find the tool message (role=tool from tool_result).
	var hasToolMessage bool
	var hasAssistantWithReasoning bool
	for _, msg := range params.Messages {
		if msg.Role == providers.RoleTool && msg.ToolCallID == "call_001" {
			hasToolMessage = true
		}
		if msg.Role == providers.RoleAssistant && msg.Reasoning != nil && msg.Reasoning.Signature == "sig_test" {
			hasAssistantWithReasoning = true
		}
	}
	require.True(t, hasToolMessage, "tool result should be converted to tool message")
	require.True(t, hasAssistantWithReasoning, "assistant thinking should preserve signature")

	// 2. Simulate ChatCompletion (as if from a provider)
	completion := &providers.ChatCompletion{
		ID:    "msg_roundtrip",
		Model: "claude-sonnet-4-20250514",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "The weather in Paris is 22°C and sunny.",
				Reasoning: &providers.Reasoning{
					Content:   "I retrieved the weather data.",
					Signature: "sig_resp",
				},
			},
			FinishReason: providers.FinishReasonStop,
		}},
		Usage: &providers.Usage{
			PromptTokens:            50,
			CompletionTokens:        15,
			TotalTokens:             65,
			CacheCreationInputTokens: 10,
			CacheReadInputTokens:    5,
			CacheCreation: &providers.CacheCreation{
				Ephemeral5mInputTokens: 8,
				Ephemeral1hInputTokens: 2,
			},
		},
	}

	// 3. Completion -> Response
	resp := FromCompletion(completion, req)
	require.NotNil(t, resp)
	require.Equal(t, "msg_roundtrip", resp.ID)
	require.Equal(t, StopReasonEndTurn, resp.StopReason)

	// Verify reasoning preserved.
	require.Len(t, resp.Content, 2)
	require.Equal(t, BlockTypeThinking, resp.Content[0].Type)
	require.Equal(t, "sig_resp", resp.Content[0].Signature)

	// Verify usage mapping.
	require.Equal(t, 50, resp.Usage.InputTokens)
	require.Equal(t, 15, resp.Usage.OutputTokens)
	require.Equal(t, 10, resp.Usage.CacheCreationInputTokens)
	require.Equal(t, 5, resp.Usage.CacheReadInputTokens)
	require.NotNil(t, resp.Usage.CacheCreation)
	require.Equal(t, 8, resp.Usage.CacheCreation.Ephemeral5mInputTokens)
	require.Equal(t, 2, resp.Usage.CacheCreation.Ephemeral1hInputTokens)
}

// Verify types are JSON-serializable.
func TestJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("MessageRequest", func(t *testing.T) {
		t.Parallel()
		req := MessageRequest{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 4096,
			System:    "You are helpful.",
			Messages:  []Message{{Role: "user", Content: "Hi"}},
		}
		b, err := json.Marshal(req)
		require.NoError(t, err)
		var restored MessageRequest
		err = json.Unmarshal(b, &restored)
		require.NoError(t, err)
		require.Equal(t, req.Model, restored.Model)
		require.Equal(t, req.MaxTokens, restored.MaxTokens)
	})

	t.Run("MessageResponse", func(t *testing.T) {
		t.Parallel()
		resp := MessageResponse{
			ID:      "msg_001",
			Type:    "message",
			Role:    "assistant",
			Model:   "claude-sonnet-4-20250514",
			Content: []ContentBlock{{Type: "text", Text: "Hello"}},
			StopReason: StopReasonEndTurn,
			Usage: Usage{InputTokens: 10, OutputTokens: 20},
		}
		b, err := json.Marshal(resp)
		require.NoError(t, err)
		var restored MessageResponse
		err = json.Unmarshal(b, &restored)
		require.NoError(t, err)
		require.Equal(t, resp.ID, restored.ID)
		require.Equal(t, resp.StopReason, restored.StopReason)
	})
}
