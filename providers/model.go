package providers

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/code-koan/llm-sdk-go/param"
)

// ModelCapabilities describes the capabilities configured on a ChatModel instance.
// Users set these at model creation time via ModelOption functional options.
// These are per-model-instance capabilities, complementary to the provider-level
// Capabilities struct.
type ModelCapabilities struct {
	Audio     bool
	Image     bool
	PDF       bool
	Reasoning bool
	Streaming bool
	Tools     bool
	Video     bool
}

// ModelOption configures a ChatModel's capabilities.
type ModelOption func(*ModelCapabilities) error

// ChatModel is a configured model instance that combines:
//   - A model ID string (e.g., "gpt-4o-audio")
//   - Model capability flags (user-configured)
//   - A reference to a Provider for executing completions
//   - A ChatBuilder factory method (NewChat())
type ChatModel struct {
	modelID      string
	capabilities ModelCapabilities
	provider     Provider
}

// ChatBuilder builds a chat completion request with fluent method chaining.
// Internal optional parameters use param.Opt[T] (inspired by anthropic-sdk-go).
type ChatBuilder struct {
	model       *ChatModel
	messages    []Message
	maxTokens   param.Opt[int]
	temperature param.Opt[float64]
	tools       []Tool
	toolChoice  any
	reasoning   ReasoningEffort
	responseFmt *ResponseFormat
	seed        param.Opt[int]
	stop        []string
	topP        param.Opt[float64]
	user        string
	stream      bool
	cacheCtrl   *CacheControlParam
	extra       map[string]any
	headers     map[string]string
}

// NewChatModel creates a ChatModel with the specified capabilities.
// provider is the Provider that will execute completions.
// modelID is the model identifier (e.g., "gpt-4o-mini").
// opts configure model-level capabilities.
// If provider implements CapabilityProvider, validates that model capabilities
// do not exceed provider capabilities.
func NewChatModel(provider Provider, modelID string, opts ...ModelOption) (*ChatModel, error) {
	mc := ModelCapabilities{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&mc); err != nil {
			return nil, err
		}
	}

	if cp, ok := provider.(CapabilityProvider); ok {
		pc := cp.Capabilities()
		if mc.Audio && !pc.CompletionAudio {
			return nil, fmt.Errorf("%s: model capability Audio requested but provider does not support audio", provider.Name())
		}
		if mc.Image && !pc.CompletionImage {
			return nil, fmt.Errorf("%s: model capability Image requested but provider does not support images", provider.Name())
		}
		if mc.Video && !pc.CompletionVideo {
			return nil, fmt.Errorf("%s: model capability Video requested but provider does not support video", provider.Name())
		}
		if mc.PDF && !pc.CompletionPDF {
			return nil, fmt.Errorf("%s: model capability PDF requested but provider does not support PDF", provider.Name())
		}
		if mc.Reasoning && !pc.CompletionReasoning {
			return nil, fmt.Errorf("%s: model capability Reasoning requested but provider does not support reasoning", provider.Name())
		}
		if mc.Streaming && !pc.CompletionStreaming {
			return nil, fmt.Errorf("%s: model capability Streaming requested but provider does not support streaming", provider.Name())
		}
		if mc.Tools && !pc.CompletionTools {
			return nil, fmt.Errorf("%s: model capability Tools requested but provider does not support tools", provider.Name())
		}
	}

	return &ChatModel{
		modelID:      modelID,
		capabilities: mc,
		provider:     provider,
	}, nil
}

// WithModelAudio enables audio input capability.
func WithModelAudio() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.Audio = true
		return nil
	}
}

// WithModelImage enables image input capability.
func WithModelImage() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.Image = true
		return nil
	}
}

// WithModelVideo enables video input capability.
func WithModelVideo() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.Video = true
		return nil
	}
}

// WithModelPDF enables PDF document input capability.
func WithModelPDF() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.PDF = true
		return nil
	}
}

// WithModelReasoning enables extended thinking/reasoning capability.
func WithModelReasoning() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.Reasoning = true
		return nil
	}
}

// WithModelStreaming enables streaming support capability.
func WithModelStreaming() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.Streaming = true
		return nil
	}
}

// WithModelTools enables tool/function calling capability.
func WithModelTools() ModelOption {
	return func(mc *ModelCapabilities) error {
		mc.Tools = true
		return nil
	}
}

// Capabilities returns the model's configured capabilities.
func (m *ChatModel) Capabilities() ModelCapabilities {
	return m.capabilities
}

// Completion executes a chat completion directly on this model.
func (m *ChatModel) Completion(ctx context.Context, params CompletionParams) (*ChatCompletion, error) {
	params.Model = m.modelID
	return m.provider.Completion(ctx, params)
}

// CompletionStream executes a streaming chat completion directly on this model.
func (m *ChatModel) CompletionStream(ctx context.Context, params CompletionParams) (<-chan ChatCompletionChunk, <-chan error) {
	params.Model = m.modelID
	params.Stream = true
	return m.provider.CompletionStream(ctx, params)
}

// ModelID returns the model identifier.
func (m *ChatModel) ModelID() string {
	return m.modelID
}

// NewChat creates a new ChatBuilder for this model.
func (m *ChatModel) NewChat() *ChatBuilder {
	return &ChatBuilder{
		model:    m,
		messages: make([]Message, 0, 4),
	}
}

// Build assembles the final CompletionParams.
// Converts Opt[T] to *T for backward compatibility.
func (b *ChatBuilder) Build() CompletionParams {
	return CompletionParams{
		CacheControl:    b.cacheCtrl,
		Extra:           b.extra,
		Headers:         b.headers,
		MaxTokens:       optToPtr(b.maxTokens),
		Messages:        b.messages,
		Model:           b.model.modelID,
		ReasoningEffort: b.reasoning,
		ResponseFormat:  b.responseFmt,
		Seed:            optToPtr(b.seed),
		Stop:            b.stop,
		Stream:          b.stream,
		Temperature:     optToPtr(b.temperature),
		ToolChoice:      b.toolChoice,
		Tools:           b.tools,
		TopP:            optToPtr(b.topP),
		User:            b.user,
	}
}

// Exec is a convenience wrapper: Build() then provider.Completion(ctx, params).
func (b *ChatBuilder) Exec(ctx context.Context) (*ChatCompletion, error) {
	return b.model.provider.Completion(ctx, b.Build())
}

// ExecStream is a convenience wrapper for streaming.
func (b *ChatBuilder) ExecStream(ctx context.Context) (<-chan ChatCompletionChunk, <-chan error) {
	params := b.Build()
	params.Stream = true
	return b.model.provider.CompletionStream(ctx, params)
}

// WithAudio appends an audio input message. Silently skipped if the model does
// not have the Audio capability configured.
func (b *ChatBuilder) WithAudio(audioData []byte, format string) *ChatBuilder {
	if !b.model.capabilities.Audio {
		return b
	}
	encoded := base64.StdEncoding.EncodeToString(audioData)
	dataURL := fmt.Sprintf("data:audio/%s;base64,%s", format, encoded)
	b.messages = append(b.messages, Message{
		Content: []ContentPart{
			{
				Type:       ContentTypeInputAudio,
				InputAudio: &InputAudio{Data: dataURL, Format: format},
			},
		},
	})
	return b
}

// WithCacheControl sets cache control parameters on the request.
func (b *ChatBuilder) WithCacheControl(cc CacheControlParam) *ChatBuilder {
	b.cacheCtrl = &cc
	return b
}

// WithExtra adds a provider-specific extra field to the request body.
func (b *ChatBuilder) WithExtra(key string, value any) *ChatBuilder {
	if b.extra == nil {
		b.extra = make(map[string]any)
	}
	b.extra[key] = value
	return b
}

// WithHeader adds a custom HTTP header to the request.
func (b *ChatBuilder) WithHeader(key, value string) *ChatBuilder {
	if b.headers == nil {
		b.headers = make(map[string]string)
	}
	b.headers[key] = value
	return b
}

// WithImage appends an image input message. Silently skipped if the model does
// not have the Image capability configured.
func (b *ChatBuilder) WithImage(imageURL string) *ChatBuilder {
	if !b.model.capabilities.Image {
		return b
	}
	b.messages = append(b.messages, Message{
		Content: []ContentPart{
			{
				Type:     ContentTypeImageURL,
				ImageURL: &ImageURL{URL: imageURL},
			},
		},
	})
	return b
}

// WithMaxTokens sets the maximum number of tokens to generate.
func (b *ChatBuilder) WithMaxTokens(n int) *ChatBuilder {
	b.maxTokens = param.NewOpt(n)
	return b
}

// WithMaxTokensOpt sets the maximum tokens using an Opt[int] value.
func (b *ChatBuilder) WithMaxTokensOpt(opt param.Opt[int]) *ChatBuilder {
	b.maxTokens = opt
	return b
}

// WithMessages appends existing messages to the conversation.
func (b *ChatBuilder) WithMessages(existing []Message) *ChatBuilder {
	b.messages = append(b.messages, existing...)
	return b
}

// WithReasoning sets the reasoning effort. Silently skipped if the model does
// not have the Reasoning capability configured.
func (b *ChatBuilder) WithReasoning(effort ReasoningEffort) *ChatBuilder {
	if !b.model.capabilities.Reasoning {
		return b
	}
	b.reasoning = effort
	return b
}

// WithResponseFormat sets the response format (e.g., JSON schema).
func (b *ChatBuilder) WithResponseFormat(rfmt ResponseFormat) *ChatBuilder {
	b.responseFmt = &rfmt
	return b
}

// WithSeed sets a seed for deterministic sampling.
func (b *ChatBuilder) WithSeed(n int) *ChatBuilder {
	b.seed = param.NewOpt(n)
	return b
}

// WithSeedOpt sets the seed using an Opt[int] value.
func (b *ChatBuilder) WithSeedOpt(opt param.Opt[int]) *ChatBuilder {
	b.seed = opt
	return b
}

// WithStop sets stop sequences.
func (b *ChatBuilder) WithStop(sequences []string) *ChatBuilder {
	b.stop = sequences
	return b
}

// WithStream enables streaming. Silently skipped if the model does not have the
// Streaming capability configured.
func (b *ChatBuilder) WithStream() *ChatBuilder {
	if !b.model.capabilities.Streaming {
		return b
	}
	b.stream = true
	return b
}

// WithSystem appends a system message.
func (b *ChatBuilder) WithSystem(text string) *ChatBuilder {
	b.messages = append(b.messages, Message{Role: RoleSystem, Content: text})
	return b
}

// WithTemperature sets the sampling temperature.
func (b *ChatBuilder) WithTemperature(t float64) *ChatBuilder {
	b.temperature = param.NewOpt(t)
	return b
}

// WithTemperatureOpt sets the temperature using an Opt[float64] value.
func (b *ChatBuilder) WithTemperatureOpt(opt param.Opt[float64]) *ChatBuilder {
	b.temperature = opt
	return b
}

// WithText appends a user text message.
func (b *ChatBuilder) WithText(text string) *ChatBuilder {
	b.messages = append(b.messages, Message{Role: RoleUser, Content: text})
	return b
}

// WithToolChoice sets the tool choice mode. Silently skipped if the model does
// not have the Tools capability configured.
func (b *ChatBuilder) WithToolChoice(choice any) *ChatBuilder {
	if !b.model.capabilities.Tools {
		return b
	}
	b.toolChoice = choice
	return b
}

// WithTools sets the available tools. Silently skipped if the model does not
// have the Tools capability configured.
func (b *ChatBuilder) WithTools(tools []Tool) *ChatBuilder {
	if !b.model.capabilities.Tools {
		return b
	}
	b.tools = tools
	return b
}

// WithTopP sets the top-p sampling parameter.
func (b *ChatBuilder) WithTopP(p float64) *ChatBuilder {
	b.topP = param.NewOpt(p)
	return b
}

// WithTopPOpt sets the top-p using an Opt[float64] value.
func (b *ChatBuilder) WithTopPOpt(opt param.Opt[float64]) *ChatBuilder {
	b.topP = opt
	return b
}

// WithUser sets the end-user identifier.
func (b *ChatBuilder) WithUser(userID string) *ChatBuilder {
	b.user = userID
	return b
}

// WithVideo appends a video input message. Silently skipped if the model does
// not have the Video capability configured.
func (b *ChatBuilder) WithVideo(videoURL string) *ChatBuilder {
	if !b.model.capabilities.Video {
		return b
	}
	b.messages = append(b.messages, Message{
		Content: []ContentPart{
			{
				Type:     ContentTypeVideoURL,
				VideoURL: &VideoURL{URL: videoURL},
			},
		},
	})
	return b
}

// optToPtr converts an Opt[T] to *T for backward compatibility with CompletionParams.
func optToPtr[T comparable](o param.Opt[T]) *T {
	if o.Valid() {
		v := o.Value
		return &v
	}
	return nil
}
