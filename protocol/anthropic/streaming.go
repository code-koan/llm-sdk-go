package anthropic

import (
	"strings"

	"github.com/code-koan/llm-sdk-go/providers"
)

// SSE event types for Anthropic streaming.
const (
	EventMessageStart      = "message_start"
	EventContentBlockStart = "content_block_start"
	EventContentBlockDelta = "content_block_delta"
	EventContentBlockStop  = "content_block_stop"
	EventMessageDelta      = "message_delta"
	EventMessageStop       = "message_stop"
	EventPing              = "ping"
)

// Delta type constants.
const (
	DeltaTypeText      = "text_delta"
	DeltaTypeThinking  = "thinking_delta"
	DeltaTypeInputJSON = "input_json_delta"
	DeltaTypeSignature = "signature_delta"
)

// StreamEvent is an Anthropic streaming SSE event.
type StreamEvent struct {
	Type  string `json:"-"`
	Index *int   `json:"index,omitempty"`

	// message_start payload
	Message *MessageResponse `json:"message,omitempty"`

	// content_block_start payload
	ContentBlock *ContentBlock `json:"content_block,omitempty"`

	// content_block_delta payload — type discriminator is delta.Type
	Delta any `json:"delta,omitempty"`

	// message_delta payload
	UsageDelta *UsageDelta `json:"usage,omitempty"`

	// error payload
	Error *ErrorDetail `json:"error,omitempty"`
}

// TextDelta is a text_delta streaming delta.
type TextDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ThinkingDelta is a thinking_delta streaming delta.
type ThinkingDelta struct {
	Type     string `json:"type"`
	Thinking string `json:"thinking"`
}

// InputJSONDelta is an input_json_delta streaming delta.
type InputJSONDelta struct {
	Type        string `json:"type"`
	PartialJSON string `json:"partial_json"`
}

// UsageDelta is the usage payload in message_delta and message_start events.
type UsageDelta struct {
	OutputTokens             int            `json:"output_tokens"`
	CacheCreationInputTokens int            `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int            `json:"cache_read_input_tokens,omitempty"`
	CacheCreation            *CacheCreation `json:"cache_creation,omitempty"`
}

// MessageDelta is the delta payload inside a message_delta event.
type MessageDelta struct {
	StopReason   string  `json:"stop_reason"`
	StopSequence *string `json:"stop_sequence"`
}

// StreamAdapter tracks state across a stream of ChatCompletionChunks and
// produces Anthropic SSE events. One instance per stream.
type StreamAdapter struct {
	messageID         string
	model             string
	content           strings.Builder
	reasoning         strings.Builder
	reasoningSig      string
	toolCalls         []toolCallAccum
	currentToolIdx    int
	inputTokens       int
	cacheCreateTokens int
	cacheReadTokens   int
	outputTokens      int

	started           bool
	reasoningStarted  bool
	reasoningFinished bool
	textBlockSeen     bool
	blocks            []*activeBlock
}

type toolCallAccum struct {
	id        string
	name      string
	arguments strings.Builder
}

type activeBlock struct {
	blockType string // "text", "thinking", or "tool_use"
	id        string
	name      string
}

// NewStreamAdapter creates a new StreamAdapter.
func NewStreamAdapter() *StreamAdapter {
	return &StreamAdapter{currentToolIdx: -1}
}

// Adapt processes a ChatCompletionChunk and returns zero or more Anthropic
// SSE events. Call for every chunk in the stream, in order.
func (a *StreamAdapter) Adapt(chunk providers.ChatCompletionChunk) []StreamEvent {
	var events []StreamEvent

	// Capture usage.
	if chunk.Usage != nil {
		a.inputTokens = chunk.Usage.PromptTokens
		a.outputTokens = chunk.Usage.CompletionTokens
		a.cacheCreateTokens = chunk.Usage.CacheCreationInputTokens
		a.cacheReadTokens = chunk.Usage.CacheReadInputTokens
	}

	if len(chunk.Choices) == 0 {
		return events
	}
	choice := chunk.Choices[0]

	// First chunk with role — emit message_start.
	if !a.started && choice.Delta.Role != "" {
		a.started = true
		if chunk.ID != "" {
			a.messageID = chunk.ID
		} else {
			a.messageID = "msg_" + chunk.ID
		}
		a.model = chunk.Model
		events = append(events, a.messageStartEvent())
	}

	// Reasoning delta.
	if choice.Delta.Reasoning != nil && choice.Delta.Reasoning.Content != "" {
		events = append(events, a.handleReasoning(choice.Delta.Reasoning.Content, choice.Delta.Reasoning.Signature)...)
	}

	// Tool call deltas.
	for _, tc := range choice.Delta.ToolCalls {
		events = append(events, a.handleToolCall(tc)...)
	}

	// Text delta.
	if choice.Delta.Content != "" {
		a.finishReasoning(&events)
		events = append(events, a.handleTextDelta(choice.Delta.Content)...)
	}

	return events
}

// Flush returns final events after the stream ends.
// Must be called exactly once after all chunks have been processed.
func (a *StreamAdapter) Flush() []StreamEvent {
	var events []StreamEvent

	if !a.started {
		return events
	}

	// Close all open blocks.
	for i := range a.blocks {
		if a.blocks[i] != nil {
			idx := i
			events = append(events, StreamEvent{
				Type:  EventContentBlockStop,
				Index: &idx,
			})
		}
	}

	// message_delta with usage.
	events = append(events, StreamEvent{
		Type: EventMessageDelta,
		Delta: MessageDelta{
			StopReason: StopReasonEndTurn,
		},
		UsageDelta: &UsageDelta{
			OutputTokens:             a.outputTokens,
			CacheCreationInputTokens: a.cacheCreateTokens,
			CacheReadInputTokens:     a.cacheReadTokens,
		},
	})

	// message_stop.
	events = append(events, StreamEvent{Type: EventMessageStop})

	return events
}

// messageStartEvent builds the initial message_start event.
func (a *StreamAdapter) messageStartEvent() StreamEvent {
	return StreamEvent{
		Type: EventMessageStart,
		Message: &MessageResponse{
			ID:      a.messageID,
			Type:    "message",
			Role:    RoleAssistant,
			Model:   a.model,
			Content: []ContentBlock{},
			Usage:   a.buildUsage(),
		},
	}
}

func (a *StreamAdapter) buildUsage() Usage {
	return Usage{
		InputTokens:              a.inputTokens,
		OutputTokens:             a.outputTokens,
		CacheCreationInputTokens: a.cacheCreateTokens,
		CacheReadInputTokens:     a.cacheReadTokens,
	}
}

func (a *StreamAdapter) handleReasoning(content, signature string) []StreamEvent {
	var events []StreamEvent
	a.reasoning.WriteString(content)
	if signature != "" {
		a.reasoningSig = signature
	}

	if !a.reasoningStarted {
		a.reasoningStarted = true
		idx := a.nextBlockIndex()
		a.blocks = append(a.blocks, &activeBlock{blockType: BlockTypeThinking})
		events = append(events, StreamEvent{
			Type:         EventContentBlockStart,
			Index:        &idx,
			ContentBlock: &ContentBlock{Type: BlockTypeThinking},
		})
	}

	idx := a.blockIndexForType(BlockTypeThinking)
	events = append(events, StreamEvent{
		Type:  EventContentBlockDelta,
		Index: &idx,
		Delta: ThinkingDelta{Type: DeltaTypeThinking, Thinking: content},
	})

	return events
}

func (a *StreamAdapter) finishReasoning(events *[]StreamEvent) {
	if !a.reasoningStarted || a.reasoningFinished {
		return
	}
	a.reasoningFinished = true
	idx := a.blockIndexForType(BlockTypeThinking)
	*events = append(*events, StreamEvent{
		Type:  EventContentBlockStop,
		Index: &idx,
	})
}

func (a *StreamAdapter) handleTextDelta(text string) []StreamEvent {
	var events []StreamEvent
	a.content.WriteString(text)

	if !a.textBlockSeen {
		a.textBlockSeen = true
		idx := a.nextBlockIndex()
		a.blocks = append(a.blocks, &activeBlock{blockType: BlockTypeText})
		events = append(events, StreamEvent{
			Type:         EventContentBlockStart,
			Index:        &idx,
			ContentBlock: &ContentBlock{Type: BlockTypeText, Text: ""},
		})
	}

	idx := a.blockIndexForType(BlockTypeText)
	events = append(events, StreamEvent{
		Type:  EventContentBlockDelta,
		Index: &idx,
		Delta: TextDelta{Type: DeltaTypeText, Text: text},
	})

	return events
}

func (a *StreamAdapter) handleToolCall(tc providers.ToolCall) []StreamEvent {
	var events []StreamEvent

	if tc.ID != "" {
		// New tool call — emit content_block_start.
		a.currentToolIdx++
		a.toolCalls = append(a.toolCalls, toolCallAccum{id: tc.ID, name: tc.Function.Name})
		idx := a.nextBlockIndex()
		a.blocks = append(a.blocks, &activeBlock{blockType: BlockTypeToolUse, id: tc.ID, name: tc.Function.Name})
		events = append(events, StreamEvent{
			Type:  EventContentBlockStart,
			Index: &idx,
			ContentBlock: &ContentBlock{
				Type: BlockTypeToolUse,
				ID:   tc.ID,
				Name: tc.Function.Name,
			},
		})
	}

	if tc.Function.Arguments != "" && a.currentToolIdx >= 0 && a.currentToolIdx < len(a.toolCalls) {
		a.toolCalls[a.currentToolIdx].arguments.WriteString(tc.Function.Arguments)
		idx := a.blockIndexForType(BlockTypeToolUse)
		// Find the correct tool_use block index (the last one).
		for i := len(a.blocks) - 1; i >= 0; i-- {
			if a.blocks[i] != nil && a.blocks[i].blockType == BlockTypeToolUse {
				idx = i
				break
			}
		}
		events = append(events, StreamEvent{
			Type:  EventContentBlockDelta,
			Index: &idx,
			Delta: InputJSONDelta{Type: DeltaTypeInputJSON, PartialJSON: tc.Function.Arguments},
		})
	}

	return events
}

// nextBlockIndex returns the next available block index.
func (a *StreamAdapter) nextBlockIndex() int {
	return len(a.blocks)
}

// blockIndexForType returns the index of the first block of the given type,
// scanning from the end for tool_use (the most recently added).
func (a *StreamAdapter) blockIndexForType(blockType string) int {
	if blockType == BlockTypeToolUse {
		for i := len(a.blocks) - 1; i >= 0; i-- {
			if a.blocks[i] != nil && a.blocks[i].blockType == BlockTypeToolUse {
				return i
			}
		}
	}
	for i, b := range a.blocks {
		if b != nil && b.blockType == blockType {
			return i
		}
	}
	return 0
}
