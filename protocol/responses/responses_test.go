package responses

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/providers"
)

func TestToCompletionParams_Simple(t *testing.T) {
	t.Parallel()

	req := &Request{
		Model: "gpt-4o-mini",
		Input: "Hello, world!",
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Equal(t, "gpt-4o-mini", params.Model)
	require.Len(t, params.Messages, 1)
	require.Equal(t, providers.RoleUser, params.Messages[0].Role)
	require.Equal(t, "Hello, world!", params.Messages[0].ContentString())
}

func TestToCompletionParams_WithInstructions(t *testing.T) {
	t.Parallel()

	req := &Request{
		Model:        "gpt-4o-mini",
		Input:        "Hello",
		Instructions: "You are a helpful assistant.",
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Messages, 2)
	require.Equal(t, providers.RoleSystem, params.Messages[0].Role)
	require.Equal(t, "You are a helpful assistant.", params.Messages[0].ContentString())
}

func TestToCompletionParams_Validation(t *testing.T) {
	t.Parallel()

	_, err := ToCompletionParams(&Request{Model: "", Input: "hi"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model")

	_, err = ToCompletionParams(&Request{Model: "gpt-4o-mini"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "input")
}

func TestToCompletionParams_WithTools(t *testing.T) {
	t.Parallel()

	req := &Request{
		Model: "gpt-4o-mini",
		Input: "What is the weather?",
		Tools: []Tool{{
			Type:        "function",
			Name:        "get_weather",
			Description: "Get weather for a location",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
		}},
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Tools, 1)
	require.Equal(t, "get_weather", params.Tools[0].Function.Name)
}

func TestToCompletionParams_WithParameters(t *testing.T) {
	t.Parallel()

	req := &Request{
		Model:           "gpt-4o-mini",
		Input:           "Hello",
		MaxOutputTokens: 500,
		Temperature:     0.7,
		TopP:            0.9,
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Equal(t, 500, *params.MaxTokens)
	require.Equal(t, 0.7, *params.Temperature)
	require.Equal(t, 0.9, *params.TopP)
}

func TestToCompletionParams_WithMultiTurnInput(t *testing.T) {
	t.Parallel()

	input := []any{
		map[string]any{"role": "user", "content": "What is the weather?"},
		map[string]any{"type": "function_call", "call_id": "call_001", "name": "get_weather", "arguments": `{"location":"Paris"}`},
		map[string]any{"type": "function_call_output", "call_id": "call_001", "output": "22°C and sunny"},
		map[string]any{"role": "user", "content": "Great, thanks!"},
	}

	req := &Request{
		Model: "gpt-4o-mini",
		Input: input,
	}

	params, err := ToCompletionParams(req)
	require.NoError(t, err)
	require.Len(t, params.Messages, 4)

	// Check function_call_output -> tool message.
	var hasToolMsg bool
	for _, msg := range params.Messages {
		if msg.Role == providers.RoleTool {
			require.Equal(t, "call_001", msg.ToolCallID)
			hasToolMsg = true
		}
	}
	require.True(t, hasToolMsg)
}

func TestFromCompletion_TextOnly(t *testing.T) {
	t.Parallel()

	completion := &providers.ChatCompletion{
		ID: "chatcmpl-123",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "Hello! How can I help?",
			},
		}},
		Usage: &providers.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	resp := FromCompletion(completion, &Request{Model: "gpt-4o-mini"})
	require.NotNil(t, resp)
	require.Equal(t, "response", resp.Object)
	require.Equal(t, StatusCompleted, resp.Status)
	require.Len(t, resp.Output, 1)
	require.Equal(t, OutputItemTypeMessage, resp.Output[0].Type)
	require.Equal(t, "assistant", resp.Output[0].Role)
	require.Len(t, resp.Output[0].Content, 1)
	require.Equal(t, ContentTypeOutputText, resp.Output[0].Content[0].Type)
	require.Equal(t, "Hello! How can I help?", resp.Output[0].Content[0].Text)
	require.NotNil(t, resp.Usage)
	require.Equal(t, 10, resp.Usage.InputTokens)
}

func TestFromCompletion_WithToolCall(t *testing.T) {
	t.Parallel()

	completion := &providers.ChatCompletion{
		ID: "chatcmpl-456",
		Choices: []providers.Choice{{
			Index: 0,
			Message: providers.Message{
				Role:    "assistant",
				Content: "",
				ToolCalls: []providers.ToolCall{{
					ID:   "call_001",
					Type: "function",
					Function: providers.FunctionCall{
						Name:      "get_weather",
						Arguments: `{"location":"Paris"}`,
					},
				}},
			},
		}},
	}

	resp := FromCompletion(completion, &Request{Model: "gpt-4o-mini"})
	require.NotNil(t, resp)
	require.Len(t, resp.Output, 1)
	require.Equal(t, OutputItemTypeFunctionCall, resp.Output[0].Type)
	require.Equal(t, "call_001", resp.Output[0].CallID)
	require.Equal(t, "get_weather", resp.Output[0].Name)
	require.Equal(t, `{"location":"Paris"}`, resp.Output[0].Arguments)
}

func TestStreamAdapter_TextOnly(t *testing.T) {
	t.Parallel()

	adapter := NewStreamAdapter()
	adapter.SetResponseID("resp_001")
	adapter.SetModel("gpt-4o-mini")

	chunks := []providers.ChatCompletionChunk{
		{ID: "resp_001", Model: "gpt-4o-mini", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{Content: "Hello"}}}},
		{ID: "resp_001", Model: "gpt-4o-mini", Choices: []providers.ChunkChoice{{Delta: providers.ChunkDelta{Content: " world"}}}},
		{ID: "resp_001", Model: "gpt-4o-mini", Usage: &providers.Usage{PromptTokens: 10, CompletionTokens: 3, TotalTokens: 13}},
	}

	var allEvents []StreamEvent
	for _, chunk := range chunks {
		allEvents = append(allEvents, adapter.Adapt(chunk)...)
	}
	allEvents = append(allEvents, adapter.Flush()...)

	// Should have: output_item.added, output_text.delta x2, response.completed
	eventTypes := make([]string, 0, len(allEvents))
	for _, e := range allEvents {
		eventTypes = append(eventTypes, e.Type)
	}
	require.Contains(t, eventTypes, EventResponseOutputItemAdded)
	require.Contains(t, eventTypes, EventResponseOutputTextDelta)
	require.Equal(t, EventResponseCompleted, eventTypes[len(eventTypes)-1])
}

func TestJSONRoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("Request", func(t *testing.T) {
		t.Parallel()
		req := Request{Model: "gpt-4o-mini", Input: "Hello"}
		b, err := json.Marshal(req)
		require.NoError(t, err)
		var restored Request
		err = json.Unmarshal(b, &restored)
		require.NoError(t, err)
		require.Equal(t, req.Model, restored.Model)
	})

	t.Run("Response", func(t *testing.T) {
		t.Parallel()
		resp := Response{
			ID:     "resp_001",
			Object: "response",
			Status: StatusCompleted,
			Model:  "gpt-4o-mini",
			Output: []OutputItem{{Type: "message", Role: "assistant", Content: []ContentPart{{Type: "output_text", Text: "Hi"}}}},
		}
		b, err := json.Marshal(resp)
		require.NoError(t, err)
		var restored Response
		err = json.Unmarshal(b, &restored)
		require.NoError(t, err)
		require.Equal(t, resp.ID, restored.ID)
	})
}
