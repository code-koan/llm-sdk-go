package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/param"

	"github.com/code-koan/llm-sdk-go/config"
	protocolanthropic "github.com/code-koan/llm-sdk-go/protocol/anthropic"
	"github.com/code-koan/llm-sdk-go/providers"
)

// Additional streaming event type constants.
const (
	eventContentBlockStopMsg = "content_block_stop"
	eventMessageStopMsg      = "message_stop"
)

// Messages implements protocol/anthropic.Provider.
// Same-protocol path: zero conversion, direct Anthropic SDK call.
func (p *Provider) Messages(
	ctx context.Context,
	req *protocolanthropic.MessageRequest,
) (*protocolanthropic.MessageResponse, error) {
	log := p.config.Logger()
	log.Debug("Messages request",
		config.Field{Key: "provider", Value: providerName},
		config.Field{Key: "model", Value: req.Model},
		config.Field{Key: "message_count", Value: len(req.Messages)},
		config.Field{Key: "has_tools", Value: len(req.Tools) > 0},
	)

	params, err := convertMessageRequest(req, p.config.Logger())
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Messages.New(ctx, params, requestOptions(nil, nil, nil)...)
	if err != nil {
		log.Debug("Messages error",
			config.Field{Key: "provider", Value: providerName},
			config.Field{Key: "model", Value: req.Model},
			config.Field{Key: "error", Value: err.Error()},
		)
		return nil, p.ConvertError(err)
	}

	result := convertMessageResponse(resp)

	log.Debug("Messages response",
		config.Field{Key: "provider", Value: providerName},
		config.Field{Key: "model", Value: result.Model},
		config.Field{Key: "stop_reason", Value: result.StopReason},
		config.Field{Key: "input_tokens", Value: result.Usage.InputTokens},
		config.Field{Key: "output_tokens", Value: result.Usage.OutputTokens},
	)

	return result, nil
}

// MessagesStream implements protocol/anthropic.Provider.
func (p *Provider) MessagesStream(
	ctx context.Context,
	req *protocolanthropic.MessageRequest,
) (<-chan protocolanthropic.StreamEvent, <-chan error) {
	events := make(chan protocolanthropic.StreamEvent)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		log := p.config.Logger()
		log.Debug("MessagesStream request",
			config.Field{Key: "provider", Value: providerName},
			config.Field{Key: "model", Value: req.Model},
			config.Field{Key: "message_count", Value: len(req.Messages)},
			config.Field{Key: "has_tools", Value: len(req.Tools) > 0},
		)

		params, err := convertMessageRequest(req, p.config.Logger())
		if err != nil {
			errs <- err
			return
		}

		stream := p.client.Messages.NewStreaming(ctx, params, requestOptions(nil, nil, nil)...)
		state := newMsgStreamState()

		//nolint:intrange // event indices from Anthropic SDK are int64.
		for stream.Next() {
			event := stream.Current()
			se, ok := convertMsgStreamEvent(event, state)
			if !ok {
				continue
			}
			select {
			case events <- se:
			case <-ctx.Done():
				return
			}
		}

		if err := stream.Err(); err != nil {
			log.Debug("MessagesStream error",
				config.Field{Key: "provider", Value: providerName},
				config.Field{Key: "model", Value: req.Model},
				config.Field{Key: "error", Value: err.Error()},
			)
			errs <- p.ConvertError(err)
			return
		}

		log.Debug("MessagesStream response",
			config.Field{Key: "provider", Value: providerName},
			config.Field{Key: "model", Value: state.model},
		)
	}()

	return events, errs
}

// msgStreamState tracks accumulated state during native Anthropic streaming.
type msgStreamState struct {
	messageID       string
	model           string
	inputTokens     int64
	cacheCreateIn   int64
	cacheReadIn     int64
}

func newMsgStreamState() *msgStreamState { return &msgStreamState{} }

// convertMsgStreamEvent converts an Anthropic streaming event to a protocolanthropic.StreamEvent.
func convertMsgStreamEvent(event anthropic.MessageStreamEventUnion, state *msgStreamState) (protocolanthropic.StreamEvent, bool) {
	switch event.Type {
	case eventMessageStart:
		msg := event.AsMessageStart()
		state.messageID = msg.Message.ID
		state.model = string(msg.Message.Model)
		state.inputTokens = msg.Message.Usage.InputTokens
		state.cacheCreateIn = msg.Message.Usage.CacheCreationInputTokens
		state.cacheReadIn = msg.Message.Usage.CacheReadInputTokens
		return protocolanthropic.StreamEvent{
			Type: protocolanthropic.EventMessageStart,
			Message: &protocolanthropic.MessageResponse{
				ID:      msg.Message.ID,
				Type:    "message",
				Role:    protocolanthropic.RoleAssistant,
				Model:   string(msg.Message.Model),
				Content: []protocolanthropic.ContentBlock{},
				Usage: protocolanthropic.Usage{
					InputTokens:              int(msg.Message.Usage.InputTokens),
					CacheCreationInputTokens: int(msg.Message.Usage.CacheCreationInputTokens),
					CacheReadInputTokens:     int(msg.Message.Usage.CacheReadInputTokens),
				},
			},
		}, true

	case eventContentBlockStart:
		block := event.AsContentBlockStart()
		return protocolanthropic.StreamEvent{
			Type:         protocolanthropic.EventContentBlockStart,
			Index:        intPtr(int(block.Index)),
			ContentBlock: convertMsgContentBlockStart(block),
		}, true

	case eventContentBlockDelta:
		delta := event.AsContentBlockDelta()
		return convertMsgContentBlockDelta(delta), true

	case eventContentBlockStopMsg:
		stop := event.AsContentBlockStop()
		return protocolanthropic.StreamEvent{
			Type:  protocolanthropic.EventContentBlockStop,
			Index: intPtr(int(stop.Index)),
		}, true

	case eventMessageDelta:
		md := event.AsMessageDelta()
		return protocolanthropic.StreamEvent{
			Type: protocolanthropic.EventMessageDelta,
			Delta: protocolanthropic.MessageDelta{
				StopReason: convertStopReason(string(md.Delta.StopReason)),
			},
			UsageDelta: &protocolanthropic.UsageDelta{
				OutputTokens: int(md.Usage.OutputTokens),
			},
		}, true

	case eventMessageStopMsg:
		return protocolanthropic.StreamEvent{Type: protocolanthropic.EventMessageStop}, true

	default:
		return protocolanthropic.StreamEvent{}, false
	}
}

// convertMsgContentBlockStart converts an Anthropic content_block_start to a protocol ContentBlock.
func convertMsgContentBlockStart(event anthropic.ContentBlockStartEvent) *protocolanthropic.ContentBlock {
	switch event.ContentBlock.Type {
	case blockTypeText:
		return &protocolanthropic.ContentBlock{Type: protocolanthropic.BlockTypeText, Text: event.ContentBlock.Text}
	case blockTypeThinking:
		return &protocolanthropic.ContentBlock{
			Type:      protocolanthropic.BlockTypeThinking,
			Thinking:  event.ContentBlock.Thinking,
			Signature: event.ContentBlock.Signature,
		}
	case blockTypeToolUse:
		return &protocolanthropic.ContentBlock{
			Type:  protocolanthropic.BlockTypeToolUse,
			ID:    event.ContentBlock.ID,
			Name:  event.ContentBlock.Name,
			Input: marshalRawJSON(event.ContentBlock.Input),
		}
	default:
		return nil
	}
}

// convertMsgContentBlockDelta converts a content_block_delta to a StreamEvent.
func convertMsgContentBlockDelta(event anthropic.ContentBlockDeltaEvent) protocolanthropic.StreamEvent {
	idx := intPtr(int(event.Index))
	switch event.Delta.Type {
	case deltaTypeText:
		return protocolanthropic.StreamEvent{
			Type:  protocolanthropic.EventContentBlockDelta,
			Index: idx,
			Delta: protocolanthropic.TextDelta{Type: protocolanthropic.DeltaTypeText, Text: event.Delta.Text},
		}
	case deltaTypeThinking:
		return protocolanthropic.StreamEvent{
			Type:  protocolanthropic.EventContentBlockDelta,
			Index: idx,
			Delta: protocolanthropic.ThinkingDelta{Type: protocolanthropic.DeltaTypeThinking, Thinking: event.Delta.Thinking},
		}
	case deltaTypeInputJSON:
		return protocolanthropic.StreamEvent{
			Type:  protocolanthropic.EventContentBlockDelta,
			Index: idx,
			Delta: protocolanthropic.InputJSONDelta{Type: protocolanthropic.DeltaTypeInputJSON, PartialJSON: event.Delta.PartialJSON},
		}
	default:
		return protocolanthropic.StreamEvent{}
	}
}

// --- Request/Response converters ---

// convertMessageRequest converts a protocol MessageRequest to Anthropic SDK params.
func convertMessageRequest(req *protocolanthropic.MessageRequest, log config.Logger) (anthropic.MessageNewParams, error) {
	messages, system := convertProtocolMessages(req.Messages, log)

	maxTokens := int64(defaultMaxTokens)
	if req.MaxTokens > 0 {
		maxTokens = int64(req.MaxTokens)
	}

	msgReq := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	if system != "" {
		msgReq.System = []anthropic.TextBlockParam{{Text: system}}
	}

	if req.Temperature != nil {
		msgReq.Temperature = anthropic.Float(*req.Temperature)
	}
	if req.TopP != nil {
		msgReq.TopP = anthropic.Float(*req.TopP)
	}
	if req.TopK != nil {
		msgReq.TopK = anthropic.Int(int64(*req.TopK))
	}
	if len(req.StopSequences) > 0 {
		msgReq.StopSequences = req.StopSequences
	}
	if len(req.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for _, t := range req.Tools {
			converted, err := convertMsgTool(t)
			if err != nil {
				return anthropic.MessageNewParams{}, err
			}
			tools = append(tools, converted)
		}
		msgReq.Tools = tools
	}
	if len(req.ToolChoice) > 0 {
		msgReq.ToolChoice = convertToolChoice(nil, nil) // default: auto
	}
	if req.Thinking != nil {
		applyMsgThinking(&msgReq, req.Thinking, maxTokens)
	}
	if req.Metadata != nil && req.Metadata.UserID != "" {
		msgReq.Metadata = anthropic.MetadataParam{UserID: param.NewOpt(req.Metadata.UserID)}
	}

	return msgReq, nil
}

// convertMessageResponse converts an Anthropic SDK response to a protocol MessageResponse.
func convertMessageResponse(resp *anthropic.Message) *protocolanthropic.MessageResponse {
	var content []protocolanthropic.ContentBlock
	for _, block := range resp.Content {
		switch block.Type {
		case blockTypeText:
			content = append(content, protocolanthropic.ContentBlock{Type: protocolanthropic.BlockTypeText, Text: block.Text})
		case blockTypeThinking:
			content = append(content, protocolanthropic.ContentBlock{
				Type:      protocolanthropic.BlockTypeThinking,
				Thinking:  block.Thinking,
				Signature: block.Signature,
			})
		case blockTypeToolUse:
			content = append(content, protocolanthropic.ContentBlock{
				Type:  protocolanthropic.BlockTypeToolUse,
				ID:    block.ID,
				Name:  block.Name,
				Input: marshalRawJSON(block.Input),
			})
		}
	}
	if content == nil {
		content = []protocolanthropic.ContentBlock{}
	}

	cacheCreate := int(resp.Usage.CacheCreationInputTokens)
	cacheRead := int(resp.Usage.CacheReadInputTokens)

	usage := protocolanthropic.Usage{
		InputTokens:              int(resp.Usage.InputTokens) + cacheCreate + cacheRead,
		OutputTokens:             int(resp.Usage.OutputTokens),
		CacheCreationInputTokens: cacheCreate,
		CacheReadInputTokens:     cacheRead,
	}
	if resp.Usage.CacheCreation.Ephemeral1hInputTokens > 0 || resp.Usage.CacheCreation.Ephemeral5mInputTokens > 0 {
		usage.CacheCreation = &protocolanthropic.CacheCreation{
			Ephemeral1hInputTokens: int(resp.Usage.CacheCreation.Ephemeral1hInputTokens),
			Ephemeral5mInputTokens: int(resp.Usage.CacheCreation.Ephemeral5mInputTokens),
		}
	}

	return &protocolanthropic.MessageResponse{
		ID:         resp.ID,
		Type:       "message",
		Role:       protocolanthropic.RoleAssistant,
		Content:    content,
		Model:      string(resp.Model),
		StopReason: convertStopReason(string(resp.StopReason)),
		Usage:      usage,
	}
}

// --- Protocol message converters ---

func convertProtocolMessages(msgs []protocolanthropic.Message, log config.Logger) ([]anthropic.MessageParam, string) {
	result := make([]anthropic.MessageParam, 0, len(msgs))
	var systemParts []string
	for _, msg := range msgs {
		if msg.Role == providers.RoleSystem {
			systemParts = append(systemParts, msgContentText(msg.Content))
			continue
		}
		if converted := convertProtocolMessage(msg, log); converted != nil {
			result = append(result, *converted)
		}
	}
	return result, strings.Join(systemParts, "\n")
}

func convertProtocolMessage(msg protocolanthropic.Message, log config.Logger) *anthropic.MessageParam {
	switch msg.Role {
	case protocolanthropic.RoleUser:
		return convertProtocolUserMessage(msg, log)
	case protocolanthropic.RoleAssistant:
		return convertProtocolAssistantMessage(msg)
	default:
		return nil
	}
}

func convertProtocolUserMessage(msg protocolanthropic.Message, log config.Logger) *anthropic.MessageParam {
	blocks, err := normalizeMsgBlocks(msg.Content)
	if err != nil || blocks == nil {
		if s, ok := msg.Content.(string); ok {
			m := anthropic.NewUserMessage(anthropic.NewTextBlock(s))
			return &m
		}
		return nil
	}
	content := make([]anthropic.ContentBlockParamUnion, 0, len(blocks))
	for _, block := range blocks {
		switch blockType(block) {
		case "text":
			text, _ := block["text"].(string)
			content = append(content, anthropic.NewTextBlock(text))
		case "image":
			if src, ok := block["source"].(map[string]any); ok {
				if imgBlock := convertMsgImageSource(src); imgBlock != nil {
					content = append(content, *imgBlock)
				}
			}
		case "tool_result":
			toolUseID, _ := block["tool_use_id"].(string)
			resultContent, _ := block["content"].(string)
			isError, _ := block["is_error"].(bool)
			content = append(content, anthropic.NewToolResultBlock(toolUseID, resultContent, isError))
		}
	}
	m := anthropic.NewUserMessage(content...)
	return &m
}

func convertProtocolAssistantMessage(msg protocolanthropic.Message) *anthropic.MessageParam {
	blocks, err := normalizeMsgBlocks(msg.Content)
	if err != nil || blocks == nil {
		if s, ok := msg.Content.(string); ok {
			m := anthropic.NewAssistantMessage(anthropic.NewTextBlock(s))
			return &m
		}
		return nil
	}
	content := make([]anthropic.ContentBlockParamUnion, 0, len(blocks))
	for _, block := range blocks {
		switch blockType(block) {
		case "text":
			text, _ := block["text"].(string)
			content = append(content, anthropic.NewTextBlock(text))
		case "thinking":
			thinking, _ := block["thinking"].(string)
			signature, _ := block["signature"].(string)
			content = append(content, anthropic.NewThinkingBlock(signature, thinking))
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			content = append(content, anthropic.NewToolUseBlock(id, block["input"], name))
		}
	}
	m := anthropic.NewAssistantMessage(content...)
	return &m
}

func convertMsgImageSource(src map[string]any) *anthropic.ContentBlockParamUnion {
	srcType, _ := src["type"].(string)
	switch srcType {
	case "base64":
		mediaType, _ := src["media_type"].(string)
		data, _ := src["data"].(string)
		if mediaType != "" && data != "" {
			block := anthropic.NewImageBlockBase64(mediaType, data)
			return &block
		}
	case "url":
		u, _ := src["url"].(string)
		if u != "" {
			block := anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: u})
			return &block
		}
	}
	return nil
}

func convertMsgTool(t protocolanthropic.Tool) (anthropic.ToolUnionParam, error) {
	schema := anthropic.ToolInputSchemaParam{Type: "object"}
	if t.InputSchema != nil {
		if t.InputSchema.Properties != nil {
			schema.Properties = t.InputSchema.Properties
		}
		if len(t.InputSchema.Required) > 0 {
			schema.Required = t.InputSchema.Required
		}
	}
	return anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        t.Name,
			Description: anthropic.String(t.Description),
			InputSchema: schema,
		},
	}, nil
}

func applyMsgThinking(req *anthropic.MessageNewParams, thinking *protocolanthropic.ThinkingConfig, maxTokens int64) {
	if thinking == nil {
		return
	}
	switch thinking.Type {
	case protocolanthropic.ThinkingTypeAuto:
		req.Thinking = anthropic.ThinkingConfigParamOfEnabled(4096)
		if maxTokens < 8192 {
			req.MaxTokens = 8192
		}
	case protocolanthropic.ThinkingTypeEnabled:
		budget := int64(thinking.BudgetTokens)
		if budget <= 0 {
			budget = 4096
		}
		req.Thinking = anthropic.ThinkingConfigParamOfEnabled(budget)
		if m := budget * 2; maxTokens < m {
			req.MaxTokens = m
		}
	}
}

// --- Shared helpers ---

// normalizeMsgBlocks normalizes message content to []map[string]any.
func normalizeMsgBlocks(content any) ([]map[string]any, error) {
	switch v := content.(type) {
	case string:
		return nil, nil
	case []any:
		blocks := make([]map[string]any, 0, len(v))
		for _, item := range v {
			b, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("expected content block object, got %T", item)
			}
			blocks = append(blocks, b)
		}
		return blocks, nil
	case []map[string]any:
		return v, nil
	default:
		return nil, fmt.Errorf("content must be a string or an array of blocks")
	}
}

func msgContentText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, item := range v {
			if b, ok := item.(map[string]any); ok && b["type"] == "text" {
				if t, ok := b["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func blockType(block map[string]any) string {
	t, _ := block["type"].(string)
	return t
}

func intPtr(v int) *int { return &v }

func marshalRawJSON(v any) json.RawMessage {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}
