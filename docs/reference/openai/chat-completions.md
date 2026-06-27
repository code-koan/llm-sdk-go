# OpenAI Chat Completions API

> 来源：https://platform.openai.com/docs/api-reference/chat/create
> 存档日期：2025-06-26

## Endpoint

POST https://api.openai.com/v1/chat/completions

Headers:
- `Content-Type: application/json`
- `Authorization: Bearer $OPENAI_API_KEY`

## Request Parameters

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 模型 ID，如 `gpt-4.1`、`gpt-4o-mini` |
| `messages` | array | ✅ | 对话消息列表 |
| `stream` | boolean | 否 | 流式响应，默认 false |
| `max_completion_tokens` | integer | 否 | 最大生成 token 数（替代已弃用的 max_tokens） |
| `temperature` | number | 否 | 采样温度 0~2，默认 1 |
| `top_p` | number | 否 | 核采样 0~1，默认 1 |
| `tools` | array | 否 | 函数调用工具定义 |
| `tool_choice` | string/object | 否 | auto / none / required / specific |
| `response_format` | object | 否 | 结构化输出：{ "type": "json_object" } 或 { "type": "json_schema", "json_schema": {...} } |

## Message Roles

| Role | 说明 |
|------|------|
| `system` | 系统级指令（传统模型） |
| `developer` | 开发者指令（o-series 模型推荐） |
| `user` | 用户消息 |
| `assistant` | 模型响应 |
| `tool` | 工具调用结果 |

## Finish Reasons

| 值 | 说明 |
|------|------|
| `stop` | 自然结束或触发了 stop 序列 |
| `length` | 达到 max_tokens 限制 |
| `tool_calls` | 模型请求工具调用 |
| `content_filter` | 内容过滤触发 |
| `function_call` | 已弃用，同 tool_calls |

## Object Types

| 值 | 说明 |
|------|------|
| `chat.completion` | 完整响应 |
| `chat.completion.chunk` | 流式 chunk |

## 示例 1：基础 Completion

### Request
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4.1",
    "messages": [
      {"role": "developer", "content": "你是一个有帮助的助手。"},
      {"role": "user", "content": "你好！"}
    ]
  }'
```

### Response
```json
{
  "id": "chatcmpl-B9MBs8CjcvOU2jLn4n570S5qMJKcT",
  "object": "chat.completion",
  "created": 1741569952,
  "model": "gpt-4.1-2025-04-14",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！我能为你提供什么帮助？",
        "refusal": null,
        "annotations": []
      },
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 19,
    "completion_tokens": 10,
    "total_tokens": 29,
    "prompt_tokens_details": {
      "cached_tokens": 0,
      "audio_tokens": 0
    },
    "completion_tokens_details": {
      "reasoning_tokens": 0,
      "audio_tokens": 0,
      "accepted_prediction_tokens": 0,
      "rejected_prediction_tokens": 0
    }
  },
  "service_tier": "default"
}
```

## 示例 2：Streaming

### Request
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4.1",
    "messages": [{"role": "user", "content": "你好！"}],
    "stream": true
  }'
```

### Response（SSE 格式，每个 chunk 一行 `data:` 前缀）
```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{"content":"你好"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1694268190,"model":"gpt-4o-mini","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]
```

Chunk 关键字段：
- 第一个 chunk：`delta.role` = "assistant"
- 中间 chunk：`delta.content` = 文本片段
- 最后一个 chunk：`delta` = {}，`finish_reason` = "stop"
- 流结束标记：`data: [DONE]`

## 示例 3：Tool Calling

### Request
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4.1",
    "messages": [{"role": "user", "content": "波士顿今天天气怎么样？"}],
    "tools": [
      {
        "type": "function",
        "function": {
          "name": "get_current_weather",
          "description": "获取指定位置的当前天气",
          "parameters": {
            "type": "object",
            "properties": {
              "location": {"type": "string", "description": "城市和州，例如 San Francisco, CA"},
              "unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
            },
            "required": ["location"]
          }
        }
      }
    ],
    "tool_choice": "auto"
  }'
```

### Response
```json
{
  "id": "chatcmpl-abc123",
  "object": "chat.completion",
  "created": 1699896916,
  "model": "gpt-4o-mini",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": null,
        "tool_calls": [
          {
            "id": "call_abc123",
            "type": "function",
            "function": {
              "name": "get_current_weather",
              "arguments": "{\"location\": \"Boston, MA\"}"
            }
          }
        ]
      },
      "finish_reason": "tool_calls"
    }
  ],
  "usage": {
    "prompt_tokens": 82,
    "completion_tokens": 17,
    "total_tokens": 99
  }
}
```

## 示例 4：Structured Output (JSON Schema)

### Request
```bash
curl https://api.openai.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o-mini",
    "response_format": {
      "type": "json_schema",
      "json_schema": {
        "strict": true,
        "name": "translation",
        "schema": {
          "type": "object",
          "properties": {
            "original_text": {"type": "string"},
            "translated_text": {"type": "string"}
          },
          "required": ["original_text", "translated_text"],
          "additionalProperties": false
        }
      }
    },
    "messages": [
      {"role": "system", "content": "翻译成英文。同时提供原文和翻译。"},
      {"role": "user", "content": "bonjour, comment ca va!"}
    ]
  }'
```

## Response Format 枚举

| `response_format.type` | 说明 |
|------------------------|------|
| `text` | 默认，普通文本 |
| `json_object` | 要求输出有效 JSON |
| `json_schema` | 严格按 JSON Schema 输出（需 `strict: true`） |

## Error Response 格式

```json
{
  "error": {
    "message": "Incorrect API key provided",
    "type": "invalid_request_error",
    "param": null,
    "code": "invalid_api_key"
  }
}
```

常见 error.type：
- `invalid_request_error` — 请求参数错误
- `authentication_error` — 认证失败
- `rate_limit_error` — 限流
- `server_error` — 服务端错误
