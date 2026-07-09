// Package anthropic provides Anthropic Messages API wire-format types and
// converters for the llm-sdk unified interface.
//
// It is transport-agnostic — no HTTP, Gin, or vendor SDK dependencies.
//
// # Types
//
// The package defines the full Anthropic Messages API request/response shape:
// MessageRequest, MessageResponse, Message, ContentBlock, Tool, Usage, etc.
//
// # Same-protocol path (zero conversion)
//
// Providers that natively support Anthropic implement the Provider interface:
//
//	ap, ok := llm.(anthropic.Provider)
//	if ok {
//	    resp, err := ap.Messages(ctx, req)       // direct, no loss
//	    events, errs := ap.MessagesStream(ctx, req)
//	}
//
// # Cross-protocol path (explicit conversion)
//
// When the resolved provider does NOT implement Provider, convert explicitly:
//
//	params, err := anthropic.ToCompletionParams(req)
//	completion, err := llm.Completion(ctx, *params)
//	resp := anthropic.FromCompletion(completion, req)
//
// # Streaming
//
// StreamAdapter converts the SDK's unified ChatCompletionChunk stream into
// Anthropic SSE events for the cross-protocol path.
package anthropic
