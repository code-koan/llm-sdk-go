---
description: Anthropic Claude provider — the reference implementation for all providers
---

# providers/anthropic

Anthropic Claude Provider — 参考实现。所有新增 Provider 以此为标准对齐。

## 核心文件

| 文件 | 职责 |
|------|------|
| `anthropic.go` | Provider 实现：构造函数 + Completion + CompletionStream + 可选接口 |
| `messages.go` | 请求/响应转换：SDK 类型 ↔ Anthropic API |
