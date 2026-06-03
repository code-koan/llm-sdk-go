# llm-sdk-go 架构

## 目录结构

```
llm-sdk-go/
├── llmsdk.go           # 根包 — 重新导出类型，简化导入
├── config/config.go    # Functional Options 配置模式
├── errors/errors.go    # 标准化错误类型 + sentinel errors
├── providers/
│   ├── types.go        # 核心接口与共享类型
│   ├── anthropic/      # Anthropic Claude (参考实现)
│   ├── openai/         # OpenAI provider
│   └── ollama/         # Ollama 本地 provider
├── internal/testutil/  # 测试工具与 fixtures
└── docs/               # 文档
```

## 核心接口 (providers/types.go)

- `Provider` — 必需: `Name()`, `Completion()`, `CompletionStream()`
- `CapabilityProvider` — 可选: `Capabilities()`
- `EmbeddingProvider` — 可选: `Embedding()`
- `ModelLister` — 可选: `ListModels()`
- `ErrorConverter` — 可选: `ConvertError()`

## 导入模式

```go
import (
    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)
```

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
