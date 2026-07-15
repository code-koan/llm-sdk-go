---
description: Tool code generation example — //llm:tool annotation usage demo
---

# examples/tools-gen

`//llm:tool` 注解与 llm-tools CLI 的使用演示。

## 核心文件

| 文件 | 职责 |
|------|------|
| `service.go` | 带 //llm:tool 注解的天气服务定义 |
| `main.go` | 使用生成的工具代码的示例入口 |
| `weather_service.gen.go` | 由 llm-tools 生成的包装器代码 |
