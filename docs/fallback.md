# Fallback Router

`fallback.Router` 将多个 LLM provider 聚合成一个节点池，提供自动 fallback、重试和选择策略。Router 自身实现 `providers.Provider` 接口，对调用方完全透明。

## 快速开始

```go
import (
    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/anthropic"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)

func main() {
    openaiProv, _ := openai.New()
    anthropicProv, _ := anthropic.New()

    // 创建 Router，默认：随机选择 + 默认重试策略。
    router, _ := llmsdk.NewRouter([]llmsdk.Provider{openaiProv, anthropicProv})

    // 用法与单个 provider 完全一致。
    resp, err := router.Completion(ctx, llmsdk.CompletionParams{
        Model:    "gpt-4o-mini",
        Messages: messages,
    })
}
```

## 配置

```go
router, err := llmsdk.NewRouter(
    providers,
    llmsdk.WithRouterSelector(llmsdk.NewRoundRobinSelector()),
    llmsdk.WithRouterRetryPolicy(llmsdk.NewDefaultRetryPolicy()),
    llmsdk.WithRouterMaxAttemptsPerProvider(3),
    llmsdk.WithLogger(myLogger),
)
```

| Option | 默认值 | 说明 |
|--------|--------|------|
| `WithRouterSelector` | `RandomSelector` | 后端选择策略 |
| `WithRouterRetryPolicy` | `DefaultRetryPolicy` | 重试策略 |
| `WithRouterMaxAttemptsPerProvider` | `2` | 每个 provider 最大尝试次数 |
| `WithLogger` | 无 | 调试日志，记录每次尝试和 fallback |

## 选择策略 (Selector)

### RandomSelector（默认）

随机选择未被排除的 provider。

```go
router, _ := llmsdk.NewRouter(providers) // 默认
```

### RoundRobinSelector

按顺序轮询。

```go
router, _ := llmsdk.NewRouter(
    providers,
    llmsdk.WithRouterSelector(llmsdk.NewRoundRobinSelector()),
)
```

### 自定义 Selector

实现 `Selector` 接口即可：

```go
type Selector interface {
    Select(providers []llmsdk.Provider, exclude map[int]struct{}) int
}
```

- `exclude` — 已失败 provider 的索引集合（O(1) 查找）
- 返回 `-1` 表示全部耗尽

## 重试策略 (RetryPolicy)

### DefaultRetryPolicy

根据错误类型自动分类：

| 错误 | 行为 |
|------|------|
| `ErrRateLimit`、`ErrModelNotFound` | 立即切换下一个 provider |
| `ErrAuthentication` | 切换下一个（不同 provider 密钥不同） |
| `ErrInvalidRequest` 等 | 切换下一个（不同 provider 可能处理不同） |
| 网络/服务器瞬态错误 | 指数退避重试同一 provider |

退避参数（可配）：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `MaxAttempts` | `5` | 每个 provider 最大尝试次数 |
| `BaseBackoff` | `1s` | 初始退避时间 |
| `MaxBackoff` | `30s` | 最大退避时间 |
| `Jitter` | `0` | 抖动系数（0.5 = 退避随机化为 [t, t*1.5)） |

```go
policy := llmsdk.NewDefaultRetryPolicy()
policy.MaxAttempts = 3
policy.BaseBackoff = 2 * time.Second
policy.Jitter = 0.3

router, _ := llmsdk.NewRouter(providers, llmsdk.WithRouterRetryPolicy(policy))
```

### 自定义 RetryPolicy

实现 `RetryPolicy` 接口：

```go
type RetryPolicy interface {
    ShouldRetry(attempt int, err error) (wait time.Duration, retry bool)
}
```

- `attempt` 从 0 开始
- `retry=false` → Router 切换下一个 provider
- `retry=true` → Router 等待 `wait` 后重试同一 provider

## 流式处理

流式 fallback 仅在首 chunk 到达前生效。一旦某个 provider 开始发送 chunk，流即建立状态，不再切换。

```go
chunks, errs := router.CompletionStream(ctx, llmsdk.CompletionParams{
    Model:    "gpt-4o-mini",
    Messages: messages,
})

for chunk := range chunks {
    fmt.Print(chunk.Choices[0].Delta.Content)
}
if err := <-errs; err != nil {
    // 所有 provider 都在首 chunk 前失败。
}
```

## Capabilities

Router 的 `Capabilities()` 返回所有后端能力的逻辑 AND。例如，如果池中有 3 个 provider，只有 2 个支持 Embedding，则 Router 报告的 `Embedding` 为 `false`。

## 错误

所有 provider 都失败时返回 `AllFailedError`：

```go
resp, err := router.Completion(ctx, params)
if err != nil {
    var allFailed *llmsdk.AllFailedError
    if errors.As(err, &allFailed) {
        fmt.Println("所有后端均失败:", allFailed.LastError)
    }
}
```

`AllFailedError.Unwrap()` 返回最后一个错误，可直接用于 `errors.Is()`。
