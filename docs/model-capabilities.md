# Model Capabilities & ChatBuilder

模型级能力配置 + ChatBuilder：能力配置 → 能力查询 → 链式构建。

## Quick Start

```go
import (
    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)

m, _ := openai.NewChatModel("gpt-4o-mini",
    llmsdk.WithModelTools(),
    llmsdk.WithModelStreaming(),
)

resp, _ := m.NewChat().
    WithSystem("You are a helpful assistant").
    WithText("Hello!").
    WithMaxTokens(1024).
    Exec(context.Background())
```

## Three-Step Flow

### Step 1: Capability Configuration

```go
m, err := openai.NewChatModel("gpt-4o-audio",
    llmsdk.WithModelAudio(),
    llmsdk.WithModelImage(),
    llmsdk.WithModelTools(),
)
```

| Option | Capability |
|--------|-----------|
| `WithModelAudio()` | Audio input |
| `WithModelImage()` | Image input |
| `WithModelVideo()` | Video input |
| `WithModelPDF()` | PDF document input |
| `WithModelReasoning()` | Extended thinking/reasoning |
| `WithModelStreaming()` | Streaming support |
| `WithModelTools()` | Tool/function calling |

If the provider does not support a requested capability, `NewChatModel` returns an error at construction time.

### Step 2: Capability Query

```go
caps := m.Capabilities()
if caps.Audio {
    // use audio features
}
```

### Step 3: Chain Building

```go
resp, err := m.NewChat().
    WithSystem("...").     // always available
    WithText("...").       // always available
    WithAudio(bytes).      // requires Audio capability
    WithImage(url).        // requires Image capability
    WithTools(tools).      // requires Tools capability
    WithMaxTokens(1024).   // always available
    Build()                // → CompletionParams
```

## ChatBuilder Method Reference

### Always Available

| Method | Description |
|--------|-------------|
| `WithSystem(text)` | Append a system message |
| `WithText(text)` | Append a user text message |
| `WithMessages([]Message)` | Load conversation history |
| `WithMaxTokens(n)` | Set max output tokens |
| `WithTemperature(t)` | Set sampling temperature |
| `WithTopP(p)` | Set nucleus sampling parameter |
| `WithSeed(n)` | Set random seed |
| `WithStop(seq)` | Set stop sequences |
| `WithUser(id)` | Set end-user identifier |
| `WithResponseFormat(f)` | Set response format (e.g., JSON Schema) |
| `WithCacheControl(cc)` | Set prompt caching |
| `WithExtra(k, v)` | Add provider-specific extra field |
| `WithHeader(k, v)` | Add custom HTTP header |

### Capability-Gated

Methods silently skipped if the capability was not configured on the ChatModel.

| Method | Required Capability |
|--------|-------------------|
| `WithAudio(data, format)` | Audio |
| `WithImage(imageURL)` | Image |
| `WithVideo(videoURL)` | Video |
| `WithTools(tools)` | Tools |
| `WithToolChoice(choice)` | Tools |
| `WithReasoning(effort)` | Reasoning |
| `WithStream()` | Streaming |

### Opt[T] Advanced Control

The SDK uses `param.Opt[T]` (inspired by anthropic-sdk-go) for optional parameter fields:

```go
// Set a value
m.NewChat().WithMaxTokens(1024)

// Explicitly set to null (JSON null)
m.NewChat().WithMaxTokensOpt(param.Null[int]())

// Check if a value was set
opt := param.NewOpt(100)
opt.Valid() // true
```

## Direct Execution

| Method | Returns |
|--------|---------|
| `Build()` | `CompletionParams` |
| `Exec(ctx)` | `(*ChatCompletion, error)` |
| `ExecStream(ctx)` | `(<-chan ChatCompletionChunk, <-chan error)` |

## Backward Compatibility

The existing `provider.Completion(ctx, params)` pattern remains unchanged. ChatBuilder is an additive API.
