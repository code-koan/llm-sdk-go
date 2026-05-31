---
description: LLM Provider 实现层 — 各厂商的 Provider 接口实现
---

# providers

多 LLM 厂商的 Provider 接口实现，每个子目录是一个独立 provider package。

## 核心文件

| 文件 | 职责 | 设计文档 |
|------|------|----------|
| `types.go` | 核心接口定义 — Provider, CapabilityProvider, EmbeddingProvider 等 | [llm-sdk-go 架构](../../../../.config/llm-sdk-go/architecture.md) |

## Provider 列表

| 目录 | Provider | 说明 |
|------|----------|------|
| `openai/` | OpenAI | OpenAI 及兼容 API（含 Compatible 模式） |
| `anthropic/` | Anthropic Claude | 参考实现 |
| `ollama/` | Ollama | 本地模型 |
| `deepseek/` | DeepSeek | 国产模型 |
| `gemini/` | Google Gemini | |
| `groq/` | Groq | 高速推理 |
| `mistral/` | Mistral | |
| `llamacpp/` | llama.cpp | 本地推理 |
| `llamafile/` | Llamafile | |
| `zai/` | Z.ai | |

## 设计文档

→ [llm-sdk-go 架构](../../../../.config/llm-sdk-go/architecture.md)
