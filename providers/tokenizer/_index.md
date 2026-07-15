---
description: Token counting — tiktoken for OpenAI, heuristic estimation for Claude/Gemini
---

# providers/tokenizer

Token 估算工具：OpenAI 模型使用 tiktoken，Claude/Gemini 使用启发式算法。

## 核心文件

| 文件 | 职责 |
|------|------|
| `tokenizer.go` | Tokenizer 接口 + 实现注册 |
| `estimator.go` | 启发式 Token 估算器（Claude/Gemini） |
