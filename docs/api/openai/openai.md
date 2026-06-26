# OpenAI Provider

> import: `"github.com/code-koan/llm-sdk-go/providers/openai"`

## 环境变量

| 变量 | 必需 | 说明 |
|------|------|------|
| `OPENAI_API_KEY` | ✅ | OpenAI API key（sk-...） |

## 快速上手

### 基础 Completion

```go
p, _ := openai.New()
resp, err := p.Completion(ctx, &providers.CompletionParams{
    Model: "gpt-4.1",
    Messages: []providers.Message{
        {Role: providers.RoleUser, Content: "Hello!"},
    },
})
fmt.Println(resp.Choices[0].Message.ContentString())
```

### Streaming

```go
stream, errs := p.CompletionStream(ctx, &providers.CompletionParams{
    Model: "gpt-4.1",
    Messages: []providers.Message{{Role: providers.RoleUser, Content: "Count to 5"}},
})
for chunk := range stream { fmt.Print(chunk.Choices[0].Delta.Content) }
if err := <-errs; err != nil { panic(err) }
```

### Tool Calling

```go
resp, err := p.Completion(ctx, &providers.CompletionParams{
    Model: "gpt-4.1",
    Messages: []providers.Message{{Role: providers.RoleUser, Content: "Weather in Tokyo?"}},
    Tools: []providers.Tool{{
        Type: "function",
        Function: providers.Function{
            Name: "get_weather", Description: "Get weather for a city",
            Parameters: map[string]any{"type": "object", "properties": map[string]any{
                "location": map[string]any{"type": "string"},
            }, "required": []string{"location"}},
        },
    }},
})
```

### Vision

```go
resp, err := p.Completion(ctx, &providers.CompletionParams{
    Model: "gpt-4o",
    Messages: []providers.Message{{
        Role: providers.RoleUser,
        Content: []providers.ContentPart{
            {Type: "text", Text: "What's in this image?"},
            {Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/photo.jpg"}},
        },
    }},
})
```

## 能力矩阵

| 能力 | 支持 | 能力 | 支持 |
|------|------|------|------|
| Completion | ✅ | CompletionStreaming | ✅ |
| CompletionImage | ✅ | CompletionTools | ✅ |
| CompletionReasoning | ✅ | CompletionPDF | ❌ |
| Embedding | ✅ | ListModels | ✅ |

## 配置选项

```go
p, _ := openai.New(
    config.WithAPIKey("sk-..."),       // 覆盖 API key
    config.WithBaseURL("https://..."), // 自定义 base URL（代理/兼容 API）
    config.WithLogger(myLogger),       // 注入 Logger
    config.WithHTTPClient(myClient),   // 自定义 HTTP client
)
```

## 错误处理

openai provider 实现了 ErrorConverter，将 OpenAI API 错误归一化为 SDK sentinel errors：
- `errors.ErrAuthentication` — API key 无效
- `errors.ErrRateLimit` — 触发限流
- `errors.ErrContextLength` — 上下文超长
- `errors.ErrContentFilter` — 内容过滤
- `errors.ErrInvalidRequest` — 请求参数错误
- `errors.ErrModelNotFound` — 模型不存在
