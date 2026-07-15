---
description: Anthropic Messages API wire-format types — transport-agnostic, same-protocol & cross-protocol paths
---

# protocol/anthropic

Anthropic Messages API 的完整请求/响应类型定义，零传输依赖。

## 两种使用路径

| 路径 | 场景 | 入口 |
|------|------|------|
| **同协议**（零转换） | Provider 原生支持 Anthropic | `anthropic.Provider` 接口 → `Messages()` / `MessagesStream()` |
| **跨协议**（显式转换） | Provider 不支持 Anthropic | `ToCompletionParams()` → SDK Completion → `FromCompletion()` |

## 核心文件

| 文件 | 职责 |
|------|------|
| `types.go` | MessageRequest, MessageResponse, ContentBlock, Tool, Usage 等 wire-format 类型 |
| `provider.go` | Anthropic-native Provider 接口定义 |
| `converters.go` | Anthropic ↔ SDK 统一接口的双向转换 |
| `hub_adapter.go` | protocol hub 适配器注册 |
| `streaming.go` | SSE 流式适配：SDK chunk → Anthropic event |
| `register.go` | Hub 注册 + 类型发现 |
| `doc.go` | 包文档 |
