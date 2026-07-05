# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.15.0] - 2026-07-05

### Added

- `tokenizer` package: local token estimation with dual strategy (tiktoken for OpenAI, character-based heuristic for Claude/Gemini)
- `Encoding` type with 7 constants: O200kBase, Cl100kBase, P50kBase, P50kEdit, R50kBase, Claude, Gemini
- `CountTokens(messages, model)` — auto-detect encoding from model name
- `CountTokensWithEncoding(messages, encoding)` — explicit encoding for user-defined models
- `CountText(text, model)` — convenience wrapper for raw text
- Re-exports in root `llmsdk` package: Encoding type, 7 encoding constants, 3 Count* functions
- Documentation: `docs/tokenizer.md`, updated `architecture.md`, `_index.md`

## [0.14.0] - 2026-07-01

### Added

- **ChatModel + ChatBuilder**: 三步一体模型级能力系统（#14）
  - `ModelOption` 功能选项: `WithModelAudio/Image/Video/PDF/Reasoning/Streaming/Tools`
  - `ChatModel`: 模型 ID + ModelCapabilities + Provider 引用 + Builder 工厂
  - `ChatBuilder`: 链式构建 `CompletionParams`，能力门控
- **param 包**: `Opt[T]` 泛型类型 — 三态可选值（omitted/null/included），对标 anthropic-sdk-go
- **ContentPart 扩展**: `InputAudio` / `VideoURL` 类型 + 4 个共享内容类型常量
- 所有 11 个 provider 新增 `NewChatModel()` 构造方法
- `examples/chat-model/` — ChatBuilder 完整使用示例
- `docs/model-capabilities.md` — ChatModel/ChatBuilder 使用文档

## [0.13.0] - 2026-07-01

### Added

- `Capabilities` struct: 新增 `CompletionAudio`、`CompletionVideo`、`TTS`、`STT` 字段，声明音视频能力
- 所有 11 个 provider 补齐完整能力声明（A-Z 排序，含 `AsyncGeneration`）
- `AsyncTaskProvider` 接口 + `AsyncTaskParams` / `AsyncTask` / `AsyncTaskStatus` 类型体系
- Seedance provider（`providers/volcengine/seedance`）：视频异步生成
- `examples/capabilities/main.go` — 能力查询示例
- `docs/providers.md` — Provider 矩阵新增 Image / Audio / Video / Async Gen 列 + Seedance 详情

### Changed

- `Capabilities` struct 字段重排为 A-Z 顺序
- `fallback.Router.Capabilities()` AND 聚合逻辑覆盖全部字段

## [0.12.0] - 2026-06-16

### Added

- Cache usage 与 rate-limit observability 文档

## [0.11.0] - 2026-06-10

### Added

- `fallback` package: multi-provider Router with automatic failover, retry, and selection strategies
- `Selector` interface + `RandomSelector` / `RoundRobinSelector` built-in implementations
- `RetryPolicy` interface + `DefaultRetryPolicy` with error classification and exponential backoff
- `Router` struct implementing `providers.Provider` as a transparent drop-in replacement
- `AllFailedError` type for inspecting failures across all providers
- Streaming fallback: retry on initial connection failure before the first chunk
- Re-exports in root `llmsdk` package: `Router`, `Selector`, `RetryPolicy`, `AllFailedError`
- Documentation: `docs/fallback.md`, updated `architecture.md`, `providers.md`, `quickstart.md`

## [0.10.0] - 2025-04-12

### Added

- Structured `Logger` interface with `Field` type in `config` package (zap-shaped, zero dependency)
- `WithLogger` functional option and `Config.Logger()` method (no-op default)
- Debug-level logging for all provider `Completion`, `CompletionStream`, `Embedding` calls
- Request logging: provider, model, message_count, has_tools, stream
- Response logging: provider, model, finish_reason, prompt_tokens, completion_tokens, total_tokens
- Error logging when API calls fail
- Re-export `Logger`, `Field`, `WithLogger` in top-level `llmsdk` package

### Changed

- `CompatibleProvider` now stores `*config.Config` for logger access
- `zai.Provider` now stores `*config.Config` for logger access
- Replaced `log.Printf` in gemini and zai providers with structured `Logger.Warn`

## [0.9.0] - 2025-04-12

### Changed

- Forked from [mozilla-ai/any-llm-go](https://github.com/mozilla-ai/any-llm-go) and rebranded to `github.com/code-koan/llm-sdk-go`
- Module path: `github.com/mozilla-ai/any-llm-go` → `github.com/code-koan/llm-sdk-go`
- Package name: `anyllm` → `llmsdk`
- Root file: `anyllm.go` → `llmsdk.go`
- Product name: `any-llm` → `llm-sdk`

### Removed

- `providers/platform/` provider and `github.com/mozilla-ai/any-llm-platform-client-go` dependency
- All `mozilla-ai` references from code and documentation

### Added

- Chinese README (`README_zh.md`) with language switch links
