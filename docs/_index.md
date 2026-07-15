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

## 文件

| 文件 | 内容 |
|------|------|
| `architecture.md` | 项目架构、核心接口、Provider 实现指南 |
| `quickstart.md` | 快速上手示例 |
| `providers.md` | Provider 实现清单与能力矩阵 |
| `fallback.md` | Router 多 Provider 负载分发与容灾 |
| `model-capabilities.md` | ChatModel + ChatBuilder 三步用法 |
| `tokenizer.md` | Token 估算 API |
| `tools.md` | Tool 代码生成 |
| `api/README.md` | API 文档概述 |
| `api/completion.md` | Completion 请求参数与响应 |
| `api/streaming.md` | 流式响应 |
| `api/errors.md` | 错误处理 |
| `api/cache-and-ratelimit.md` | Cache + Rate Limit |
| `api/anthropic/anthropic.md` | Anthropic Provider SDK 用法 |
| `api/openai/openai.md` | OpenAI Provider SDK 用法 |
| `api/volcengine/seedance.md` | 火山方舟 Seedance API 用法 |
| `reference/anthropic/` | Anthropic 官方 API 参考存档 |
| `reference/openai/` | OpenAI 官方 API 参考存档 |
| `reference/volcengine/` | 火山方舟 官方 API 参考存档 |
