---
description: OpenAI Responses API wire-format types — transport-agnostic, Chat Completions conversion path
---

# protocol/responses

OpenAI Responses API（Codex CLI 主用）的请求/响应类型定义，零传输依赖。

> 当前无 Provider 原生实现 Responses API，统一走 Chat Completions 转换路径。

## 核心文件

| 文件 | 职责 |
|------|------|
| `types.go` | ResponseRequest, Response, ResponseItem, Tool 等 wire-format 类型 |
| `converters.go` | Response ↔ Chat Completions 双向转换 |
| `hub_adapter.go` | protocol hub 适配器注册 |
| `streaming.go` | SSE 流式适配：SDK chunk → Response event |
| `register.go` | Hub 注册 + 类型发现 |
| `doc.go` | 包文档 |
