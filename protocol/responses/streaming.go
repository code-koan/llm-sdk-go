package responses

import "github.com/code-koan/llm-sdk-go/providers"

// SSE event type constants for Responses API streaming.
const (
	EventResponseCreated                    = "response.created"
	EventResponseOutputItemAdded            = "response.output_item.added"
	EventResponseContentPartAdded           = "response.content_part.added"
	EventResponseOutputTextDelta            = "response.output_text.delta"
	EventResponseOutputTextDone             = "response.output_text.done"
	EventResponseFunctionCallArgumentsDelta = "response.function_call_arguments.delta"
	EventResponseFunctionCallArgumentsDone  = "response.function_call_arguments.done"
	EventResponseOutputItemDone             = "response.output_item.done"
	EventResponseCompleted                  = "response.completed"
)

// StreamEvent is a Responses API SSE event.
type StreamEvent struct {
	Type string `json:"-"`
	Data any    `json:"-"`
}

// ResponseCreatedEvent is the response.created event payload.
type ResponseCreatedEvent struct {
	ID     string `json:"id"`
	Object string `json:"object"`
	Status string `json:"status"`
}

// OutputItemAddedEvent is the response.output_item.added event payload.
type OutputItemAddedEvent struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

// OutputTextDeltaEvent is the response.output_text.delta event payload.
type OutputTextDeltaEvent struct {
	Delta string `json:"delta"`
}

// FunctionCallArgumentsDeltaEvent is the response.function_call_arguments.delta event payload.
type FunctionCallArgumentsDeltaEvent struct {
	Delta string `json:"delta"`
}

// ResponseCompletedEvent is the response.completed event payload.
type ResponseCompletedEvent struct {
	ID     string           `json:"id"`
	Object string           `json:"object"`
	Status string           `json:"status"`
	Model  string           `json:"model"`
	Output []map[string]any `json:"output,omitempty"`
	Usage  map[string]int   `json:"usage,omitempty"`
}

// StreamAdapter converts ChatCompletionChunk stream to Responses API SSE events.
type StreamAdapter struct {
	respID     string
	model      string
	content    string
	msgEmitted bool

	// Tool call accumulation.
	toolCalls []toolAccum
	curTool   *toolAccum

	inputTokens  int
	outputTokens int
	totalTokens  int
}

type toolAccum struct {
	id        string
	name      string
	arguments string
}

// NewStreamAdapter creates a new StreamAdapter.
func NewStreamAdapter() *StreamAdapter {
	return &StreamAdapter{}
}

// Adapt processes a ChatCompletionChunk and returns zero or more Responses
// API SSE events. Call for every chunk in the stream, in order.
func (a *StreamAdapter) Adapt(chunk providers.ChatCompletionChunk) []StreamEvent {
	var events []StreamEvent

	if chunk.Usage != nil {
		a.inputTokens = chunk.Usage.PromptTokens
		a.outputTokens = chunk.Usage.CompletionTokens
		a.totalTokens = chunk.Usage.TotalTokens
	}

	if len(chunk.Choices) == 0 {
		return events
	}
	choice := chunk.Choices[0]

	// Tool call handling.
	for _, tc := range choice.Delta.ToolCalls {
		if tc.ID != "" {
			a.curTool = &toolAccum{id: tc.ID, name: tc.Function.Name}
			a.toolCalls = append(a.toolCalls, *a.curTool)
			events = append(events, StreamEvent{
				Type: EventResponseOutputItemAdded,
				Data: OutputItemAddedEvent{
					Type: "function_call",
					ID:   tc.ID,
					Name: tc.Function.Name,
				},
			})
		}
		if tc.Function.Arguments != "" && a.curTool != nil {
			a.curTool.arguments += tc.Function.Arguments
			events = append(events, StreamEvent{
				Type: EventResponseFunctionCallArgumentsDelta,
				Data: FunctionCallArgumentsDeltaEvent{Delta: tc.Function.Arguments},
			})
		}
	}

	// Text delta.
	if choice.Delta.Content != "" {
		if !a.msgEmitted {
			a.msgEmitted = true
			events = append(events, StreamEvent{
				Type: EventResponseOutputItemAdded,
				Data: OutputItemAddedEvent{
					Type: "message",
					ID:   a.respID + "_msg",
				},
			})
		}
		a.content += choice.Delta.Content
		events = append(events, StreamEvent{
			Type: EventResponseOutputTextDelta,
			Data: OutputTextDeltaEvent{Delta: choice.Delta.Content},
		})
	}

	return events
}

// SetResponseID sets the response identifier for the stream.
func (a *StreamAdapter) SetResponseID(id string) {
	a.respID = id
}

// SetModel sets the model identifier for the stream.
func (a *StreamAdapter) SetModel(model string) {
	a.model = model
}

// Flush returns final events after the stream ends.
func (a *StreamAdapter) Flush() []StreamEvent {
	var outputItems []map[string]any
	if a.content != "" {
		outputItems = append(outputItems, map[string]any{
			"type": "message",
			"role": "assistant",
			"content": []map[string]any{
				{"type": "output_text", "text": a.content},
			},
		})
	}
	for _, acc := range a.toolCalls {
		outputItems = append(outputItems, map[string]any{
			"type":      "function_call",
			"id":        acc.id,
			"call_id":   acc.id,
			"name":      acc.name,
			"arguments": acc.arguments,
		})
	}

	usage := map[string]int{
		"input_tokens":  a.inputTokens,
		"output_tokens": a.outputTokens,
		"total_tokens":  a.totalTokens,
	}

	return []StreamEvent{{
		Type: EventResponseCompleted,
		Data: ResponseCompletedEvent{
			ID:     a.respID,
			Object: "response",
			Status: "completed",
			Model:  a.model,
			Output: outputItems,
			Usage:  usage,
		},
	}}
}
