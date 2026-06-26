# Anthropic Provider

> import: `"github.com/code-koan/llm-sdk-go/providers/anthropic"`

## 环境变量

| 变量 | 必需 | 说明 |
|------|------|------|
| `ANTHROPIC_API_KEY` | ✅ | Anthropic API key |
| `ANTHROPIC_BASE_URL` | 否 | 自定义 base URL |

## 快速上手

### 基础 Completion
```go
resp, err := p.Completion(ctx, providers.CompletionParams{
    Model: "claude-sonnet-4-6",
    Messages: []providers.Message{
        {Role: providers.RoleSystem, Content: "You are a helpful assistant."},
        {Role: providers.RoleUser, Content: "Hello!"},
    },
})
fmt.Println(resp.Choices[0].Message.ContentString())
```
### Streaming
```go
chunks, errs := p.CompletionStream(ctx, providers.CompletionParams{
    Model: "claude-sonnet-4-6",
    Messages: []providers.Message{
        {Role: providers.RoleUser, Content: "Count to 5"},
    },
})
for chunk := range chunks { fmt.Print(chunk.Choices[0].Delta.Content) }
if err := <-errs; err != nil { panic(err) }
```
### Tool Use
```go
resp, err := p.Completion(ctx, providers.CompletionParams{
    Model: "claude-sonnet-4-6",
    Messages: []providers.Message{
        {Role: providers.RoleUser, Content: "What's the weather in Paris?"},
    },
    Tools: []providers.Tool{{
        Type: "function",
        Function: providers.Function{
            Name: "get_weather", Description: "Get weather",
            Parameters: map[string]any{
                "type": "object",
                "properties": map[string]any{"location": map[string]any{"type": "string"}},
                "required": []string{"location"},
            },
        },
    }},
})
```
### Extended Thinking
```go
resp, err := p.Completion(ctx, providers.CompletionParams{
    Model: "claude-sonnet-4-6", ReasoningEffort: providers.ReasoningEffortHigh,
    Messages: []providers.Message{
        {Role: providers.RoleUser, Content: "Solve a complex problem"},
    },
})
```
### Prompt Caching
```go
resp, err := p.Completion(ctx, providers.CompletionParams{
    Model: "claude-sonnet-4-6",
    Messages: []providers.Message{
        {Role: providers.RoleSystem, Content: systemPrompt,
            CacheControl: &providers.CacheControlParam{Type: "ephemeral", TTL: "5m"}},
        {Role: providers.RoleUser, Content: "Hello"},
    },
})
```

## Anthropic vs OpenAI 关键差异

| 特性 | Anthropic | OpenAI |
|------|-----------|--------|
| System Prompt | SDK 从 Messages 提取 RoleSystem 转顶层 system | messages 内 role: "system" |
| Tool 定义 | name + input_schema (JSON Schema) | function.name + function.parameters |
| Tool 响应 | type: "tool_use", input 为 JSON object | type: "function", arguments 为 JSON string |
| Streaming | 6 种 SSE 事件 | data: 行 + [DONE] |
| Thinking | auto/low/medium/high 映射 token budget | reasoning_effort (low/medium/high) |
| Temperature | Opus 4.7+ 不支持 | 0~2，默认 1 |

## 能力矩阵
Completion / Streaming / Image / PDF / Tools / Reasoning = ✅
Embedding / ListModels = ❌

## 配置选项

```go
p, _ := anthropic.New(
    config.WithAPIKey("sk-ant-..."), config.WithBaseURL("https://..."),
    config.WithLogger(myLogger), config.WithHTTPClient(myClient),
    config.WithUserID("user-123"),
)
```

## 错误处理

| 错误类型 | 触发条件 |
|----------|----------|
| ErrAuthentication | API key 无效 |
| ErrRateLimit | 触发限流 / 服务过载 |
| ErrContextLength | 上下文超长 (HTTP 400 + context_length) |
| ErrContentFilter | 内容安全拒绝 |
| ErrInvalidRequest | 请求参数错误 (HTTP 400) |
| ErrModelNotFound | 模型不存在 (HTTP 404) |
| ErrProvider | 其他 provider 错误 |

## Stop Reasons
| Anthropic | SDK FinishReason |
|-----------|-----------------|
| end_turn | stop |
| max_tokens | length |
| stop_sequence | stop |
| tool_use | tool_calls |
