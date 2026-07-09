package anthropic

import (
	"context"

	"github.com/code-koan/llm-sdk-go/providers"
)

// Provider is an optional interface for providers that natively support the
// Anthropic Messages API. Same-protocol calls bypass CompletionParams entirely,
// achieving zero-loss forward and backward conversion.
type Provider interface {
	providers.Provider
	Messages(ctx context.Context, req *MessageRequest) (*MessageResponse, error)
	MessagesStream(ctx context.Context, req *MessageRequest) (<-chan StreamEvent, <-chan error)
}
