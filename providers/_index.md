---
description: LLM Provider 实现层 — 各厂商的 Provider 接口实现
---

# providers

多 LLM 厂商的 Provider 接口实现，每个子目录是一个独立 provider package。

## 核心文件

| 文件 | 职责 | 设计文档 |
|------|------|----------|
| `types.go` | 核心接口定义 — Provider, CapabilityProvider, EmbeddingProvider 等 | [架构](../docs/architecture.md) |

## Provider 列表

| 目录 | 说明 |
|------|------|
| `anthropic/` | Anthropic Claude — 参考实现 |
| `openai/` | OpenAI + Compatible 包装器 |
| `deepseek/` | DeepSeek |
| `gemini/` | Google Gemini |
| `groq/` | Groq |
| `mistral/` | Mistral AI |
| `ollama/` | Ollama 本地模型 |
| `llamacpp/` | llama.cpp |
| `llamafile/` | Llamafile |
| `volcengine/` | 火山方舟 |
| `zai/` | Z.AI |
| `tokenizer/` | Token 估算工具 |

## 设计文档

→ [架构](../docs/architecture.md)
→ [Provider 文档](../docs/providers.md)
