# Cache 数据透传 & Rate Limit 增强

## 问题

当前 SDK 在两个关键维度缺失数据透传能力：

### Cache 数据全部丢弃

| 上游 | 返回数据 | SDK 现状 |
|------|---------|----------|
| Anthropic | `usage.cache_read_input_tokens` / `cache_creation_input_tokens` / `cache_creation.ephemeral_1h/5m` | **丢弃** |
| OpenAI | `usage.prompt_tokens_details.cached_tokens` | **丢弃** |
| Anthropic 请求 | `cache_control: {type:"ephemeral", ttl:"1h"}` — 2026 年默认 TTL 已从 1h 降为 5m，需显式设置 | **不支持** |

`Usage` 只有 `PromptTokens` / `CompletionTokens` / `TotalTokens` / `ReasoningTokens`。

### Rate Limit 不可观测

| 能力 | 现状 |
|------|------|
| `RateLimitError.RetryAfter` | 结构体有字段，**从未填充**（两上游 SDK 的 Error 暴露 `Response *http.Response`，可提取） |
| `X-RateLimit-*` header | 未读取 |
| 配置级 user 隔离 | 只能每请求手动设 `params.User` |

## 设计方案

分 4 个阶段，全部向后兼容（`omitempty` 字段）：

### Phase A: Cache 响应侧

`Usage` 新增字段：

```go
type Usage struct {
    // ... existing ...
    CacheReadInputTokens    int            `json:"cache_read_input_tokens,omitempty"`
    CacheCreationInputTokens int           `json:"cache_creation_input_tokens,omitempty"`
    CacheCreation           *CacheCreation `json:"cache_creation,omitempty"`
}

type CacheCreation struct {
    Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens,omitempty"`
    Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens,omitempty"`
}
```

- Anthropic 原生映射
- OpenAI `cached_tokens` → `CacheReadInputTokens`
- 涉及：`types.go`、`anthropic/anthropic.go`、`openai/compatible.go`、`llmsdk.go`

### Phase B: Cache 请求控制

`ContentPart` 新增 `cache_control`：

```go
type ContentPart struct {
    // ... existing ...
    CacheControl *CacheControlParam `json:"cache_control,omitempty"`
}

type CacheControlParam struct {
    Type string `json:"type"`           // "ephemeral"
    TTL  string `json:"ttl,omitempty"`  // "5m" | "1h"
}
```

Anthropic provider 映射 `CacheControl` → `anthropic.CacheControlEphemeralParam`。

### Phase C: Rate Limit 增强

各 provider 从 `apiErr.Response.Header.Get("Retry-After")` 填充 `RateLimitError.RetryAfter`：

```go
case 429:
    rateErr := errors.NewRateLimitError(providerName, err)
    if apiErr.Response != nil {
        if ra := apiErr.Response.Header.Get("Retry-After"); ra != "" {
            if s, _ := strconv.Atoi(ra); s > 0 {
                rateErr.RetryAfter = s
            }
        }
    }
    return rateErr
```

涉及：`errors/errors.go`、`openai/compatible.go`、`anthropic/anthropic.go`、`zai/zai.go`。

### Phase D: WithExtra 请求体注入 & WithUserID

核心思路：**HTTP Transport 层 deep-merge `Config.Extra` 到 JSON 请求体**，零侵入所有 provider。

```go
// extraFieldsRoundTripper 拦截所有 JSON 请求，deep-merge Config.Extra
type extraFieldsRoundTripper struct {
    base  http.RoundTripper
    extra map[string]any
}
```

调用方：

```go
provider, _ := deepseek.New(
    deepseek.WithAPIKey("sk-..."),
    deepseek.WithExtra("user_id", "tenant-abc"),  // 注入任意 JSON 字段
)
```

`WithUserID` 是对 `WithExtra` 的语义化封装，同时设置 `DefaultUser` 供 provider 层的 `params.User` 路径使用。

## API 示例

```go
// 1. 使用 WithUserID 隔离
provider, _ := deepseek.New(
    deepseek.WithAPIKey("sk-..."),
    deepseek.WithUserID("tenant-abc"),
)

// 2. 使用 cache_control（Anthropic）
msgs := []llmsdk.Message{{
    Role: llmsdk.RoleUser,
    Content: []llmsdk.ContentPart{{
        Type:         "text",
        Text:         "large reference document...",
        CacheControl: &llmsdk.CacheControlParam{Type: "ephemeral", TTL: "1h"},
    }, {
        Type: "text",
        Text: "user question",
    }},
}}

// 3. 读取 cache 命中
resp, _ := provider.Completion(ctx, params)
fmt.Println("cache hit:", resp.Usage.CacheReadInputTokens)
fmt.Println("cache write:", resp.Usage.CacheCreationInputTokens)

// 4. Rate limit 重试
if errors.Is(err, llmsdk.ErrRateLimit) {
    var rateErr *llmsdk.RateLimitError
    errors.As(err, &rateErr)
    time.Sleep(time.Duration(rateErr.RetryAfter) * time.Second)
}
```

## 兼容性

- 所有新增字段均为 `omitempty`，向后兼容
- `ContentPart.CacheControl` 仅 Anthropic provider 消费，其他 provider 忽略
- Transport 层的 extra body 注入仅在 `Config.Extra` 非空时激活

## 不改的边界

- 不实现自动重试（调用方职责）
- 不实现请求队列/令牌桶（上层网关能力）
- OpenAI 请求侧 cache_control（OpenAI 自动缓存）
- DeepSeek context caching（不同机制）

---

详细实现方案见 `.claude/plans/floating-sprouting-swing.md`
