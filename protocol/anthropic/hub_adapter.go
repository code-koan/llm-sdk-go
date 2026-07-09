package anthropic

import "github.com/code-koan/llm-sdk-go/providers"

// hubStreamAdapter wraps the Anthropic StreamAdapter to satisfy protocol.StreamAdapter.
type hubStreamAdapter struct {
	inner *StreamAdapter
}

func (a *hubStreamAdapter) Adapt(chunk providers.ChatCompletionChunk) []any {
	events := a.inner.Adapt(chunk)
	result := make([]any, len(events))
	for i, e := range events {
		result[i] = e
	}
	return result
}

func (a *hubStreamAdapter) Flush() []any {
	events := a.inner.Flush()
	result := make([]any, len(events))
	for i, e := range events {
		result[i] = e
	}
	return result
}
