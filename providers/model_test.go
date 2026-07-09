package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/param"
)

// ── Inline mock (avoids import cycle: testutil imports providers) ────────────

type mockProvider struct {
	name          string
	caps          Capabilities
	compCalls     []CompletionParams
	compStrmCalls []CompletionParams
}

func (m *mockProvider) Name() string               { return m.name }
func (m *mockProvider) Capabilities() Capabilities { return m.caps }
func (m *mockProvider) Completion(_ context.Context, p CompletionParams) (*ChatCompletion, error) {
	m.compCalls = append(m.compCalls, p)
	return &ChatCompletion{Model: p.Model}, nil
}

func (m *mockProvider) CompletionStream(
	_ context.Context,
	p CompletionParams,
) (<-chan ChatCompletionChunk, <-chan error) {
	m.compStrmCalls = append(m.compStrmCalls, p)
	ch := make(chan ChatCompletionChunk)
	errCh := make(chan error, 1)
	close(ch)
	close(errCh)
	return ch, errCh
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func newTestProvider(t *testing.T, name string, caps Capabilities) *mockProvider {
	t.Helper()
	return &mockProvider{name: name, caps: caps}
}

// allCapabilities returns a Capabilities struct with all completion features enabled.
func allCapabilities() Capabilities {
	return Capabilities{
		Completion:          true,
		CompletionAudio:     true,
		CompletionImage:     true,
		CompletionPDF:       true,
		CompletionReasoning: true,
		CompletionStreaming: true,
		CompletionTools:     true,
		CompletionVideo:     true,
	}
}

// ---------------------------------------------------------------------------
// ChatModel Tests
// ---------------------------------------------------------------------------

func TestNewChatModel_basic(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)
	require.Equal(t, "test-model", m.ModelID())
	require.Equal(t, ModelCapabilities{}, m.Capabilities())
}

func TestNewChatModel_withCapabilities(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", allCapabilities())
	m, err := NewChatModel(p, "test-model",
		WithModelAudio(), WithModelImage(), WithModelVideo(),
		WithModelPDF(), WithModelReasoning(), WithModelStreaming(), WithModelTools())
	require.NoError(t, err)

	caps := m.Capabilities()
	require.True(t, caps.Audio)
	require.True(t, caps.Image)
	require.True(t, caps.Video)
	require.True(t, caps.PDF)
	require.True(t, caps.Reasoning)
	require.True(t, caps.Streaming)
	require.True(t, caps.Tools)
}

func TestNewChatModel_validationAgainstProvider(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	_, err := NewChatModel(p, "test-model", WithModelAudio())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support audio")

	_, err = NewChatModel(p, "test-model", WithModelVideo())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support video")

	_, err = NewChatModel(p, "test-model", WithModelImage())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support images")

	_, err = NewChatModel(p, "test-model", WithModelPDF())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support PDF")

	_, err = NewChatModel(p, "test-model", WithModelReasoning())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support reasoning")

	_, err = NewChatModel(p, "test-model", WithModelStreaming())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support streaming")

	_, err = NewChatModel(p, "test-model", WithModelTools())
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support tools")
}

func TestNewChatModel_validationPasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		opt    ModelOption
		setter func(c *Capabilities)
	}{
		{name: "audio", opt: WithModelAudio(), setter: func(c *Capabilities) { c.CompletionAudio = true }},
		{name: "image", opt: WithModelImage(), setter: func(c *Capabilities) { c.CompletionImage = true }},
		{name: "video", opt: WithModelVideo(), setter: func(c *Capabilities) { c.CompletionVideo = true }},
		{name: "pdf", opt: WithModelPDF(), setter: func(c *Capabilities) { c.CompletionPDF = true }},
		{name: "reasoning", opt: WithModelReasoning(), setter: func(c *Capabilities) { c.CompletionReasoning = true }},
		{name: "streaming", opt: WithModelStreaming(), setter: func(c *Capabilities) { c.CompletionStreaming = true }},
		{name: "tools", opt: WithModelTools(), setter: func(c *Capabilities) { c.CompletionTools = true }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caps := Capabilities{}
			tc.setter(&caps)
			p := newTestProvider(t, "test", caps)
			m, err := NewChatModel(p, "test-model", tc.opt)
			require.NoError(t, err)
			require.NotNil(t, m)
		})
	}
}

func TestChatModel_Capabilities(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", allCapabilities())
	m, err := NewChatModel(p, "test-model",
		WithModelAudio(), WithModelImage(), WithModelVideo(),
		WithModelPDF(), WithModelReasoning(), WithModelStreaming(), WithModelTools())
	require.NoError(t, err)

	caps := m.Capabilities()
	require.True(t, caps.Audio)
	require.True(t, caps.Image)
	require.True(t, caps.Video)
	require.True(t, caps.PDF)
	require.True(t, caps.Reasoning)
	require.True(t, caps.Streaming)
	require.True(t, caps.Tools)
}

func TestChatModel_Completion(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	result, err := m.Completion(context.Background(), CompletionParams{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	require.NoError(t, err)
	require.Equal(t, "test-model", result.Model)

	require.Len(t, p.compCalls, 1)
	require.Equal(t, "test-model", p.compCalls[0].Model)
	require.Len(t, p.compCalls[0].Messages, 1)
}

func TestChatModel_CompletionStream(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	ch, errCh := m.CompletionStream(context.Background(), CompletionParams{
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	for range ch {
	}
	streamErr := <-errCh
	require.NoError(t, streamErr)

	require.Len(t, p.compStrmCalls, 1)
	require.Equal(t, "test-model", p.compStrmCalls[0].Model)
	require.True(t, p.compStrmCalls[0].Stream)
	require.Len(t, p.compStrmCalls[0].Messages, 1)
}

// ---------------------------------------------------------------------------
// ChatBuilder Tests
// ---------------------------------------------------------------------------

func TestChatBuilder_WithSystem(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithSystem("system message")
	require.Len(t, b.messages, 1)
	require.Equal(t, RoleSystem, b.messages[0].Role)
	require.Equal(t, "system message", b.messages[0].ContentString())
}

func TestChatBuilder_WithText(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithText("user message")
	require.Len(t, b.messages, 1)
	require.Equal(t, RoleUser, b.messages[0].Role)
	require.Equal(t, "user message", b.messages[0].ContentString())
}

func TestChatBuilder_WithMessages(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	existing := []Message{
		{Role: RoleUser, Content: "msg1"},
		{Role: RoleAssistant, Content: "msg2"},
	}
	b := m.NewChat().WithSystem("sys").WithMessages(existing)
	require.Len(t, b.messages, 3)
	require.Equal(t, "sys", b.messages[0].ContentString())
	require.Equal(t, "msg1", b.messages[1].ContentString())
	require.Equal(t, "msg2", b.messages[2].ContentString())
}

func TestChatBuilder_WithMaxTokens(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithMaxTokens(100)
	require.True(t, b.maxTokens.Valid())
	require.Equal(t, 100, b.maxTokens.Value)
}

func TestChatBuilder_WithTemperature(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithTemperature(0.7)
	require.True(t, b.temperature.Valid())
	require.Equal(t, 0.7, b.temperature.Value)
}

func TestChatBuilder_chainMultipleMessages(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithSystem("sys").WithText("user1").WithText("user2")
	require.Len(t, b.messages, 3)
	require.Equal(t, RoleSystem, b.messages[0].Role)
	require.Equal(t, "sys", b.messages[0].ContentString())
	require.Equal(t, RoleUser, b.messages[1].Role)
	require.Equal(t, "user1", b.messages[1].ContentString())
	require.Equal(t, RoleUser, b.messages[2].Role)
	require.Equal(t, "user2", b.messages[2].ContentString())
}

func TestChatBuilder_Build_completeParams(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", allCapabilities())
	m, err := NewChatModel(p, "test-model",
		WithModelTools(), WithModelReasoning(), WithModelStreaming())
	require.NoError(t, err)

	params := m.NewChat().
		WithSystem("sys").
		WithText("user").
		WithMaxTokens(100).
		WithTemperature(0.7).
		WithTopP(0.9).
		WithSeed(42).
		WithStop([]string{"stop1"}).
		WithUser("user-123").
		WithResponseFormat(ResponseFormat{Type: "json_object"}).
		WithCacheControl(CacheControlParam{Type: "ephemeral"}).
		WithExtra("k1", "v1").
		WithHeader("X-Custom", "val").
		WithToolChoice(ToolChoice{Type: "function", Function: &ToolChoiceFunction{Name: "get_weather"}}).
		WithReasoning(ReasoningEffortHigh).
		WithStream().
		Build()

	require.Equal(t, "test-model", params.Model)
	require.Len(t, params.Messages, 2)

	require.NotNil(t, params.MaxTokens)
	require.Equal(t, 100, *params.MaxTokens)

	require.NotNil(t, params.Temperature)
	require.Equal(t, 0.7, *params.Temperature)

	require.NotNil(t, params.TopP)
	require.Equal(t, 0.9, *params.TopP)

	require.NotNil(t, params.Seed)
	require.Equal(t, 42, *params.Seed)

	require.Equal(t, []string{"stop1"}, params.Stop)
	require.Equal(t, "user-123", params.User)

	require.NotNil(t, params.ResponseFormat)
	require.Equal(t, "json_object", params.ResponseFormat.Type)

	require.NotNil(t, params.CacheControl)
	require.Equal(t, CacheControlType("ephemeral"), params.CacheControl.Type)

	require.Equal(t, "v1", params.Extra["k1"])
	require.Equal(t, "val", params.Headers["X-Custom"])

	require.NotNil(t, params.ToolChoice)
	require.Equal(t, ReasoningEffortHigh, params.ReasoningEffort)
	require.True(t, params.Stream)
}

func TestChatBuilder_Build_emptyBuilder(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	params := m.NewChat().Build()
	require.Equal(t, "test-model", params.Model)
	require.Empty(t, params.Messages)
}

func TestChatBuilder_Build_OptToPtr(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	params := m.NewChat().Build()
	require.Nil(t, params.MaxTokens)
	require.Nil(t, params.Temperature)
	require.Nil(t, params.TopP)
	require.Nil(t, params.Seed)
}

func TestChatBuilder_WithAudio_enabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{CompletionAudio: true})
	m, err := NewChatModel(p, "test-model", WithModelAudio())
	require.NoError(t, err)

	b := m.NewChat().WithAudio([]byte("test-data"), "wav")
	require.Len(t, b.messages, 1)

	parts := b.messages[0].ContentParts()
	require.Len(t, parts, 1)
	require.Equal(t, ContentTypeInputAudio, parts[0].Type)
	require.NotNil(t, parts[0].InputAudio)
	require.Equal(t, "wav", parts[0].InputAudio.Format)
	require.True(t, strings.HasPrefix(parts[0].InputAudio.Data, "data:audio/wav;base64,"))
}

func TestChatBuilder_WithAudio_disabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithAudio([]byte("test-data"), "wav")
	require.Empty(t, b.messages)
}

func TestChatBuilder_WithImage_enabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{CompletionImage: true})
	m, err := NewChatModel(p, "test-model", WithModelImage())
	require.NoError(t, err)

	b := m.NewChat().WithImage("https://example.com/img.png")
	require.Len(t, b.messages, 1)

	parts := b.messages[0].ContentParts()
	require.Len(t, parts, 1)
	require.Equal(t, ContentTypeImageURL, parts[0].Type)
	require.NotNil(t, parts[0].ImageURL)
	require.Equal(t, "https://example.com/img.png", parts[0].ImageURL.URL)
}

func TestChatBuilder_WithImage_disabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithImage("https://example.com/img.png")
	require.Empty(t, b.messages)
}

func TestChatBuilder_WithVideo_enabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{CompletionVideo: true})
	m, err := NewChatModel(p, "test-model", WithModelVideo())
	require.NoError(t, err)

	b := m.NewChat().WithVideo("https://example.com/video.mp4")
	require.Len(t, b.messages, 1)

	parts := b.messages[0].ContentParts()
	require.Len(t, parts, 1)
	require.Equal(t, ContentTypeVideoURL, parts[0].Type)
	require.NotNil(t, parts[0].VideoURL)
	require.Equal(t, "https://example.com/video.mp4", parts[0].VideoURL.URL)
}

func TestChatBuilder_WithVideo_disabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithVideo("https://example.com/video.mp4")
	require.Empty(t, b.messages)
}

func TestChatBuilder_WithTools_enabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{CompletionTools: true})
	m, err := NewChatModel(p, "test-model", WithModelTools())
	require.NoError(t, err)

	tools := []Tool{
		{Type: "function", Function: Function{Name: "get_weather", Description: "Get the weather"}},
	}
	b := m.NewChat().WithTools(tools)
	require.Len(t, b.tools, 1)
	require.Equal(t, "get_weather", b.tools[0].Function.Name)
}

func TestChatBuilder_WithTools_disabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	tools := []Tool{
		{Type: "function", Function: Function{Name: "get_weather"}},
	}
	b := m.NewChat().WithTools(tools)
	require.Nil(t, b.tools)
}

func TestChatBuilder_WithReasoning_enabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{CompletionReasoning: true})
	m, err := NewChatModel(p, "test-model", WithModelReasoning())
	require.NoError(t, err)

	b := m.NewChat().WithReasoning(ReasoningEffortHigh)
	require.Equal(t, ReasoningEffortHigh, b.reasoning)
}

func TestChatBuilder_WithReasoning_disabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithReasoning(ReasoningEffortHigh)
	require.Equal(t, ReasoningEffort(""), b.reasoning)
}

func TestChatBuilder_WithStream_enabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{CompletionStreaming: true})
	m, err := NewChatModel(p, "test-model", WithModelStreaming())
	require.NoError(t, err)

	b := m.NewChat().WithStream()
	require.True(t, b.stream)
}

func TestChatBuilder_WithStream_disabled(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	b := m.NewChat().WithStream()
	require.False(t, b.stream)
}

func TestChatBuilder_Exec(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	result, err := m.NewChat().WithText("hello").Exec(context.Background())
	require.NoError(t, err)
	require.Equal(t, "test-model", result.Model)

	require.Len(t, p.compCalls, 1)
	require.Equal(t, "test-model", p.compCalls[0].Model)
	require.Len(t, p.compCalls[0].Messages, 1)
	require.Equal(t, "hello", p.compCalls[0].Messages[0].ContentString())
}

func TestChatBuilder_ExecStream(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	ch, errCh := m.NewChat().WithText("hello").ExecStream(context.Background())
	for range ch {
	}
	streamErr := <-errCh
	require.NoError(t, streamErr)

	require.Len(t, p.compStrmCalls, 1)
	require.Equal(t, "test-model", p.compStrmCalls[0].Model)
	require.True(t, p.compStrmCalls[0].Stream)
	require.Len(t, p.compStrmCalls[0].Messages, 1)
	require.Equal(t, "hello", p.compStrmCalls[0].Messages[0].ContentString())
}

func TestChatBuilder_WithMaxTokensOpt(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	params := m.NewChat().WithMaxTokensOpt(param.Null[int]()).Build()
	require.Nil(t, params.MaxTokens)
}

func TestChatBuilder_WithExtra(t *testing.T) {
	t.Parallel()

	p := newTestProvider(t, "test", Capabilities{})
	m, err := NewChatModel(p, "test-model")
	require.NoError(t, err)

	params := m.NewChat().
		WithExtra("key1", "val1").
		WithExtra("key2", 42).
		Build()

	require.Equal(t, "val1", params.Extra["key1"])
	require.Equal(t, 42, params.Extra["key2"])
}
