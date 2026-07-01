# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
