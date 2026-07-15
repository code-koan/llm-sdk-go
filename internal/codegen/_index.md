---
description: Internal code generation — parses //llm:tool annotations, generates tool function wrappers
---

# internal/codegen

解析 Go 源文件中的 `//llm:tool` 注解，生成工具函数包装器。仅限内部使用。

## 核心文件

| 文件 | 职责 |
|------|------|
| `parse.go` | Go 源文件解析：提取 //llm:tool 注解 + 函数签名 |
| `schema.go` | JSON Schema 生成 |
| `emit.go` | 代码生成：输出 .gen.go 包装器文件 |
