# llm-sdk-go 架构

## 目录结构

```
llm-sdk-go/
├── llmsdk.go           # 根包 — 重新导出类型，简化导入
├── config/config.go    # Functional Options 配置模式
├── errors/errors.go    # 标准化错误类型 + sentinel errors
├── param/              # 泛型参数封装 — Opt[T] 三态可选值
│   ├── param.go        # Opt[T] 泛型类型
│   └── wrappers.go     # 便捷构造函数 (Int/Float/Bool/String)
├── fallback/           # Fallback Router — 多后端聚合、重试、选择策略
│   ├── fallback.go     # Router (实现 Provider 接口)
│   ├── selectors.go    # Selector 接口 + Random/RoundRobin 实现
│   └── retry.go        # RetryPolicy 接口 + DefaultRetryPolicy
├── providers/
│   ├── types.go        # 核心接口与共享类型
│   ├── anthropic/      # Anthropic Claude (参考实现)
│   ├── openai/         # OpenAI provider
│   ├── ollama/         # Ollama 本地 provider
│   └── tokenizer/      # Token 估算 — tiktoken (OpenAI) + 启发式 (Claude/Gemini)
├── internal/testutil/  # 测试工具与 fixtures
└── docs/               # 文档
```

## 核心接口 (providers/types.go)

- `Provider` — 必需: `Name()`, `Completion()`, `CompletionStream()`
- `CapabilityProvider` — 可选: `Capabilities()`
- `EmbeddingProvider` — 可选: `Embedding()`
- `ModelLister` — 可选: `ListModels()`
- `ErrorConverter` — 可选: `ConvertError()`
- `AsyncTaskProvider` — 可选: `SubmitTask()`, `GetTask()`

## Model 层 (providers/model.go)

ChatModel 是 Provider 之上的模型级能力配置层，提供三步一体 API：

```
ChatModel (模型 ID + ModelCapabilities + Provider 引用)
  ├── Capabilities() → 能力查询
  └── NewChat() → ChatBuilder (链式构建)
        ├── WithSystem/WithText/... → 消息构建
        ├── WithAudio/WithImage/... → 能力门控
        ├── Build() → CompletionParams
        └── Exec()/ExecStream() → 直接执行
```

`param.Opt[T]` (inspired by anthropic-sdk-go) 统一了可选参数的三态表示（omitted/null/included）。

详见 [model-capabilities.md](model-capabilities.md)。

## 导入模式

```go
import (
    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)
```

## Fallback Router (fallback/)

`Router` 自身实现 `providers.Provider`，对调用方完全透明。内部通过 `Selector`（选择策略）和 `RetryPolicy`（重试策略）两个接口解耦：

```
调用方 → Router (impl Provider)
           ├── Selector — 从池中选出下一个 provider
           ├── RetryPolicy — 决定是否重试/等待多久
           └── []providers.Provider — 后端池
```

- 非流式：完整 fallback + retry 循环
- 流式：首 chunk 到达前可 fallback；首 chunk 后不再切换
- 详见 [fallback.md](fallback.md)

## 错误处理

标准化错误: `ErrRateLimit`, `ErrAuthentication`, `ErrContextLength`, `ErrContentFilter`, `ErrModelNotFound`, `ErrInvalidRequest`, `ErrMissingAPIKey`。

Provider 通过 `errors.As` 将厂商特定错误转换为 SDK 类型错误。

## Logger

- `Logger` 接口: `Debug()`, `Info()`, `Warn()`, `Error()` 带 `...Field`
- 默认 no-op（不使用 `WithLogger` 时零开销）
- 所有 provider API 调用在 Debug 级别记录请求参数和响应

## 添加新 Provider

1. `providers/{name}/{name}.go` 实现 `Provider` 接口
2. 按需实现可选接口 (`ErrorConverter` 等)
3. 添加测试（`t.Parallel()`）
4. 参考 `providers/anthropic/` 作为标准示例
