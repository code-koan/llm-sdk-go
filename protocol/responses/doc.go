// Package responses provides OpenAI Responses API wire-format types and
// converters for the llm-sdk unified interface.
//
// The Responses API is a newer OpenAI endpoint primarily used by Codex CLI.
// This package is transport-agnostic — no HTTP or vendor SDK dependencies.
//
// Unlike Anthropic, there is no native "ResponsesProvider" interface yet,
// since no llm-sdk-go provider implements the Responses API natively.
// Use ToCompletionParams and FromCompletion for the Chat Completions
// conversion path, and StreamAdapter for streaming.
package responses
