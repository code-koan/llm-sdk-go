[English](README.md) | [中文](README_zh.md)

<div align="center">

# llm-sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/code-koan/llm-sdk-go.svg)](https://pkg.go.dev/github.com/code-koan/llm-sdk-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/code-koan/llm-sdk-go)](https://goreportcard.com/report/github.com/code-koan/llm-sdk-go)

![Go 1.25+](https://img.shields.io/badge/go-1.25%2B-blue.svg)

**使用统一接口与任意 LLM 提供商通信。**

在 OpenAI、Anthropic、DeepSeek、Mistral、Ollama 等之间切换，无需修改代码。

[文档](docs/) | [示例](examples/) | [贡献指南](CONTRIBUTING.md)

</div>

## 快速开始

### 安装

```bash
go get github.com/code-koan/llm-sdk-go
# 设置 API Key
export OPENAI_API_KEY="YOUR_KEY_HERE"  # 或 ANTHROPIC_API_KEY 等
```

### 发送对话请求

```go
package main

import (
    "context"
    "fmt"
    "log"

    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)

func main() {
    ctx := context.Background()

    provider, err := openai.New()
    if err != nil {
        log.Fatal(err)
    }

    response, err := provider.Completion(ctx, llmsdk.CompletionParams{
        Model: "gpt-4o-mini",
        Messages: []llmsdk.Message{
            {Role: llmsdk.RoleUser, Content: "你好！"},
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(response.Choices[0].Message.Content)
}
```

**就这样！** 切换提供商只需更改 import 和构造函数（例如用 `anthropic.New()` 替代 `openai.New()`）。

## 安装详情

### 环境要求

- Go 1.25 或更高版本
- 所需 LLM 提供商的 API Key

导入主包和你需要的提供商：

```go
import (
    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"    // OpenAI
    "github.com/code-koan/llm-sdk-go/providers/anthropic" // Anthropic
)
```

查看[支持的提供商列表](docs/providers.md)选择你需要的提供商。

### 配置 API Key

为所选提供商设置环境变量：

```bash
export OPENAI_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"
export DEEPSEEK_API_KEY="your-key-here"
# ... 等等
```

也可以在代码中直接传入 API Key：

```go
provider, err := openai.New(llmsdk.WithAPIKey("your-key-here"))
```

## 为什么选择 `llm-sdk-go`？

- **简洁统一的接口** - 所有提供商使用相同的类型和模式
- **惯用 Go 风格** - 遵循 Go 规范，正确处理错误和 context
- **使用官方提供商 SDK** - 底层使用 `github.com/openai/openai-go` 和 `github.com/anthropics/anthropic-sdk-go`
- **类型安全** - 所有请求和响应类型均有完整类型定义
- **流式支持** - 基于 channel 的流式响应，符合 Go 习惯
- **久经验证的模式** - 跨多个 LLM 提供商的统一接口设计

## 使用方法

创建提供商实例并发送请求：

```go
import (
    "context"
    "fmt"
    "log"

    llmsdk "github.com/code-koan/llm-sdk-go"
    "github.com/code-koan/llm-sdk-go/providers/openai"
)

// 创建一次，多次复用。
provider, err := openai.New(llmsdk.WithAPIKey("your-api-key"))
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "你好！"},
    },
})
if err != nil {
    log.Fatal(err)
}

fmt.Println(response.Choices[0].Message.Content)
```

提供商实例可复用，推荐在生产环境中使用。

### 流式响应

使用 channel 接收流式响应：

```go
chunks, errs := provider.CompletionStream(ctx, llmsdk.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "写一首关于 Go 语言的短诗。"},
    },
})

for chunk := range chunks {
    if len(chunk.Choices) > 0 {
        fmt.Print(chunk.Choices[0].Delta.Content)
    }
}

if err := <-errs; err != nil {
    log.Fatal(err)
}
```

### 工具调用 / 函数调用

```go
response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4o-mini",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "巴黎的天气怎么样？"},
    },
    Tools: []llmsdk.Tool{
        {
            Type: "function",
            Function: llmsdk.Function{
                Name:        "get_weather",
                Description: "获取指定城市的当前天气",
                Parameters: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "location": map[string]any{
                            "type":        "string",
                            "description": "城市名称",
                        },
                    },
                    "required": []string{"location"},
                },
            },
        },
    },
    ToolChoice: "auto",
})

// 检查工具调用。
if len(response.Choices[0].Message.ToolCalls) > 0 {
    tc := response.Choices[0].Message.ToolCalls[0]
    fmt.Printf("函数: %s, 参数: %s\n", tc.Function.Name, tc.Function.Arguments)
}
```

### 扩展思考（推理）

对于支持扩展思考的模型（如 Claude）：

```go
response, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "claude-sonnet-4-20250514",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "请逐步解答：80 的 15% 是多少？"},
    },
    ReasoningEffort: llmsdk.ReasoningEffortMedium,
})

if response.Choices[0].Message.Reasoning != nil {
    fmt.Println("思考过程:", response.Choices[0].Message.Reasoning.Content)
}
fmt.Println("答案:", response.Choices[0].Message.Content)
```

### 错误处理

所有提供商的错误都被归一化为通用错误类型：

```go
response, err := provider.Completion(ctx, params)
if err != nil {
    switch {
    case errors.Is(err, llmsdk.ErrRateLimit):
        // 处理限流 - 可使用退避重试。
    case errors.Is(err, llmsdk.ErrAuthentication):
        // 处理认证错误 - 检查 API Key。
    case errors.Is(err, llmsdk.ErrContextLength):
        // 处理上下文过长 - 减少输入。
    default:
        // 处理其他错误。
    }
}
```

也可以使用类型断言获取更多详情：

```go
var rateLimitErr *llmsdk.RateLimitError
if errors.As(err, &rateLimitErr) {
    fmt.Printf("被 %s 限流: %s\n", rateLimitErr.Provider, rateLimitErr.Message)
}
```

### 查找可用模型

每个提供商使用自己的模型标识符。查找可用模型：
- 查看提供商的文档
- 使用 `ListModels` API（如果提供商支持）：

```go
provider, _ := openai.New()
models, err := provider.ListModels(ctx)
for _, model := range models.Data {
    fmt.Println(model.ID)
}
```

## 支持的提供商

|   提供商   | 对话补全 | 流式输出 | 工具调用 | 推理思考 | 向量嵌入 |
|:----------:|:--------:|:--------:|:--------:|:--------:|:--------:|
| Anthropic  |    ✅    |    ✅    |    ✅    |    ✅    |    ❌    |
|  DeepSeek  |    ✅    |    ✅    |    ✅    |    ✅    |    ❌    |
|   Gemini   |    ✅    |    ✅    |    ✅    |    ✅    |    ✅    |
|    Groq    |    ✅    |    ✅    |    ✅    |    ❌    |    ❌    |
|  llama.cpp |    ✅    |    ✅    |    ✅    |    ❌    |    ✅    |
| Llamafile  |    ✅    |    ✅    |    ✅    |    ❌    |    ✅    |
|  Mistral   |    ✅    |    ✅    |    ✅    |    ✅    |    ✅    |
|   Ollama   |    ✅    |    ✅    |    ✅    |    ✅    |    ✅    |
|   OpenAI   |    ✅    |    ✅    |    ✅    |    ✅    |    ✅    |
|    z.ai    |    ✅    |    ✅    |    ✅    |    ✅    |    ❌    |

更多提供商即将支持！完整列表请查看 [docs/providers.md](docs/providers.md)。

## 文档

- **[快速入门](docs/quickstart.md)** - 快速上手
- **[支持的提供商](docs/providers.md)** - 所有支持的 LLM 提供商列表
- **[API 参考](docs/api/)** - 完整的 API 文档
- **[示例代码](examples/)** - 常见用例的代码示例

## 贡献

欢迎各水平的开发者贡献代码！请阅读[贡献指南](CONTRIBUTING.md)或提交 Issue 讨论变更。

## 许可证

本项目基于 Apache License 2.0 许可 - 详见 [LICENSE](LICENSE) 文件。
