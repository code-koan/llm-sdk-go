# Completion API

The completion API is the primary way to interact with LLM providers.

## Quick Start

```go
import (
    "context"

    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)

provider, err := openai.New()
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "Hello!"},
    },
})
```

## Provider Interface

### `Completion`

```go
func (p *Provider) Completion(
    ctx context.Context,
    params CompletionParams,
) (*ChatCompletion, error)
```

Performs a chat completion request.

**Parameters:**
- `ctx` - Context for cancellation and timeouts
- `params` - Completion parameters including model and messages

**Returns:**
- `*ChatCompletion` - The completion response
- `error` - Any error that occurred

**Example:**

```go
response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "claude-3-5-haiku-latest",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleSystem, Content: "You are a helpful assistant."},
        {Role: llmsdk.RoleUser, Content: "What is Go?"},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Choices[0].Message.Content)
```

## CompletionParams

Full parameters for completion requests:

```go
type CompletionParams struct {
    // CacheControl sets top-level cache control for providers that support it.
    CacheControl *CacheControlParam `json:"cache_control,omitempty"`

    // Extra adds non-conflicting provider-specific JSON body fields for this request.
    Extra map[string]any `json:"-"`

    // Headers adds provider-specific request headers for this request.
    Headers map[string]string `json:"-"`

    // MaxTokens limits the response length.
    MaxTokens *int `json:"max_tokens,omitempty"`

    // Messages is the conversation history (required).
    Messages []Message `json:"messages"`

    // Model is the model ID to use (required).
    Model string `json:"model"`

    // OverrideBody explicitly overrides generated JSON body fields for this request.
    OverrideBody map[string]any `json:"-"`

    // ParallelToolCalls allows multiple tool calls in one response.
    ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`

    // ResponseFormat specifies the output format.
    ResponseFormat *ResponseFormat `json:"response_format,omitempty"`

    // ReasoningEffort controls extended thinking (for supported models).
    ReasoningEffort ReasoningEffort `json:"reasoning_effort,omitempty"`

    // Seed for deterministic outputs (if supported).
    Seed *int `json:"seed,omitempty"`

    // Stop sequences that will halt generation.
    Stop []string `json:"stop,omitempty"`

    // Stream enables streaming responses.
    Stream bool `json:"stream,omitempty"`

    // Temperature controls randomness (0.0-2.0, default varies by provider).
    Temperature *float64 `json:"temperature,omitempty"`

    // ToolChoice controls tool selection behavior.
    // Can be "auto", "none", "required", or a ToolChoice struct.
    ToolChoice any `json:"tool_choice,omitempty"`

    // Tools available for the model to call.
    Tools []Tool `json:"tools,omitempty"`

    // TopP controls nucleus sampling (0.0-1.0).
    TopP *float64 `json:"top_p,omitempty"`

    // User identifier for tracking. Overrides WithUserID for this request.
    User string `json:"user,omitempty"`
}
```

## Message Types

### Basic Message

```go
type Message struct {
    Role       string      `json:"role"`
    Content    any         `json:"content"` // string or []ContentPart
    Name       string      `json:"name,omitempty"`
    ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
    ToolCallID string      `json:"tool_call_id,omitempty"`
    Reasoning  *Reasoning  `json:"reasoning,omitempty"`
}
```

### Role Constants

```go
const (
    RoleSystem    = "system"
    RoleUser      = "user"
    RoleAssistant = "assistant"
    RoleTool      = "tool"
)
```

### Multimodal Content

For messages with images or other content types:

```go
message := llmsdk.Message{
    Role: llmsdk.RoleUser,
    Content: []llmsdk.ContentPart{
        {Type: "text", Text: "What's in this image?"},
        {Type: "image_url", ImageURL: &llmsdk.ImageURL{
            URL: "https://example.com/image.jpg",
        }},
    },
}
```

## Cache Control and Per-Request Extensions

### Anthropic Cache Control

Anthropic supports explicit prompt caching through `CacheControl` on `CompletionParams`, `ContentPart`, and `Tool`. The default Anthropic ephemeral TTL is 5 minutes; set `CacheControlTTL1h` for the optional 1 hour TTL.

```go
response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "claude-sonnet-4-20250514",
    Messages: []llmsdk.Message{{
        Role: llmsdk.RoleUser,
        Content: []llmsdk.ContentPart{{
            Type: "text",
            Text: "large stable context...",
            CacheControl: &llmsdk.CacheControlParam{
                Type: llmsdk.CacheControlTypeEphemeral,
                TTL:  llmsdk.CacheControlTTL1h,
            },
        }},
    }},
})
```

OpenAI-compatible providers use upstream automatic prompt caching when available; the SDK maps returned cache usage but does not send OpenAI cache-control request fields.

### Headers, Extra, and OverrideBody

Use per-request escape hatches for provider-specific options:

```go
response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4.1",
    Messages: messages,
    Headers: map[string]string{
        "OpenAI-Project": "proj_xxx",
    },
    Extra: map[string]any{
        "prompt_cache_key": "tenant-abc-docs-v1",
    },
    OverrideBody: map[string]any{
        "temperature": 0.2,
    },
})
```

`Extra` only adds non-conflicting JSON body fields. `OverrideBody` is for explicit overrides. The SDK does not perform transport-level deep merge.

### Default and Per-Request User IDs

`llmsdk.WithUserID` sets a provider default user identifier. `CompletionParams.User` overrides it for one request. OpenAI-compatible providers send `user`; Anthropic sends `metadata.user_id`.

```go
provider, _ := openai.New(
    llmsdk.WithAPIKey("sk-..."),
    llmsdk.WithUserID("tenant-abc"),
)

response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4.1",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "Hello"},
    },
    User: "tenant-override", // optional per-request override
})
```

## Response Types

### ChatCompletion

```go
type ChatCompletion struct {
    ID                string   `json:"id"`
    Object            string   `json:"object"`
    Created           int64    `json:"created"`
    Model             string   `json:"model"`
    Choices           []Choice `json:"choices"`
    Usage             *Usage   `json:"usage,omitempty"`
    SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}
```

### Choice

```go
type Choice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason string  `json:"finish_reason,omitempty"`
}
```

### Finish Reasons

```go
const (
    FinishReasonStop          = "stop"
    FinishReasonLength        = "length"
    FinishReasonToolCalls     = "tool_calls"
    FinishReasonContentFilter = "content_filter"
)
```

## Tool Calling

### Defining Tools

```go
tools := []llmsdk.Tool{
    {
        Type: "function",
        Function: llmsdk.Function{
            Name:        "get_weather",
            Description: "Get the current weather for a location",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{
                    "location": map[string]any{
                        "type":        "string",
                        "description": "City name",
                    },
                },
                "required": []string{"location"},
            },
        },
    },
}
```

### Processing Tool Calls

```go
response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model:    "gpt-4o-mini",
    Messages: messages,
    Tools:    tools,
})

if response.Choices[0].FinishReason == llmsdk.FinishReasonToolCalls {
    for _, tc := range response.Choices[0].Message.ToolCalls {
        // Process tool call.
        result := executeFunction(tc.Function.Name, tc.Function.Arguments)

        // Add tool result to messages.
        messages = append(messages, response.Choices[0].Message)
        messages = append(messages, llmsdk.Message{
            Role:       llmsdk.RoleTool,
            Content:    result,
            ToolCallID: tc.ID,
        })
    }

    // Continue conversation with tool results.
    response, err = provider.Completion(ctx, llmsdk.CompletionParams{
        Model:    "gpt-4o-mini",
        Messages: messages,
        Tools:    tools,
    })
}
```

## See Also

- [Streaming](streaming.md) - Streaming responses
- [Errors](errors.md) - Error handling
