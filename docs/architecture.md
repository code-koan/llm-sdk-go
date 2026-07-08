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
│   ├── openai/         # OpenAI + OpenAI-compatible 协议
│   │   └── extra.go     #   ExtraFields 提取 helper
│   ├── ollama/         # Ollama 本地 provider
│   ├── zai/            # z.ai provider
│   ├── gemini/         # Gemini provider
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
3. 在 `providers/types.go` 对应字段注释中补充协议映射
4. 添加测试（`t.Parallel()`）
5. 参考 `providers/anthropic/` 作为标准示例

## 协议映射表

`providers/types.go` 是跨协议字段的唯一真相源。每个字段注释中包含「协议映射」段落，说明该字段在各 Provider 中的对应概念。

以下为当前跨协议字段的完整矩阵：

| Canonical 字段 | Anthropic | OpenAI / Compatible | zai | Gemini |
|---|---|---|---|---|
| `Message.Content` | `TextBlock.Text` | `content` | `content` | `Part.Text` |
| `Message.Reasoning` | `ThinkingBlock` | `reasoning_content` (ExtraFields) | `reasoning_content` (平铺) | `Part.Thought.Text` (delta) |
| `Message.ToolCalls` | `ToolUseBlock` | `tool_calls` | `tool_calls` | `Part.FunctionCall` |
| `Message.ToolCallID` | `ToolResultBlock.ToolUseID` | `tool_call_id` | — | `Part.FunctionResponse` |
| `Reasoning.Content` | `ThinkingBlock.Thinking` | `reasoning_content` | `reasoning_content` | `Part.Text` |
| `Reasoning.Signature` | `ThinkingBlock.Signature` | — | — | — |
| `Usage.ReasoningTokens` | 不单独报告 | `CompletionTokensDetails.ReasoningTokens` | `usage.reasoning_tokens` | `UsageMetadata.ThoughtsTokenCount` |
| `ToolCall.Extra` | — | — | — | `ThoughtSignature` |
| `ChunkDelta.Reasoning` | `ThinkingDelta` | `reasoning_content` (delta ExtraFields) | `reasoning_content` (delta) | `Part.Thought` |

### Reasoning Round-Trip 兼容矩阵

多轮对话中，同 Provider reasoning 可完整 round-trip；
跨 Provider 受上游协议约束：

| 上游 ↓ / 下游 → | Anthropic | OpenAI Compatible | zai |
|---|---:|---:|---:|
| **Anthropic** | ✅ Signature 存在 | ✅ 只传 Content | ✅ 只传 Content |
| **OpenAI Compatible** | ❌ 缺 Signature，跳过 | ✅ | ✅ |
| **zai** | ❌ 缺 Signature，跳过 | ✅ | ✅ |

### 新增跨协议字段流程

1. `providers/types.go` 对应 struct 加字段 + 协议映射注释
2. 各 Provider 响应转换：原生格式 → 规范字段
3. 各 Provider 请求转换：规范字段 → 原生格式
4. `llmsdk.go` re-export（如有必要）
5. 本文档更新映射表
