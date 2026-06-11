# llm-sdk-go

多 LLM 厂商统一 SDK — 用单一接口对接 OpenAI、Anthropic、DeepSeek、Mistral、Ollama 等，响应统一为 OpenAI 格式。

## 项目概要

- **仓库**: [code-koan/llm-sdk-go](https://github.com/code-koan/llm-sdk-go)
- **语言**: Go 1.25+
- **核心接口**: `Provider`（`Completion` / `CompletionStream` / `Embedding` / `ListModels`）
- **设计模式**: Functional Options 配置、标准化错误类型、可插拔 Logger

## 核心能力

- 屏蔽多厂商 API 差异，统一调用方式
- 流式响应统一为 OpenAI SSE 格式
- 结构化错误归一化（限流、认证、上下文超长等）
- 零依赖 Logger 接口，默认 no-op

## 在 monorepo 中的位置

被 `aix` 作为 LLM 调用层依赖，是所有 Go 服务访问 LLM 的唯一入口。

## 文件

- [architecture.md](architecture.md) — 项目架构、核心接口、Provider 实现指南
- [api/cache-and-ratelimit.md](api/cache-and-ratelimit.md) — **待实现**：Cache 数据透传 & Rate Limit 增强方案
