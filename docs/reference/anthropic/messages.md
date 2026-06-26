# Anthropic Messages API

> 来源：https://docs.anthropic.com/en/api/messages
> 存档日期：2025-06-26

## Endpoint

POST https://api.anthropic.com/v1/messages

Headers:
- `Content-Type: application/json`
- `x-api-key: $ANTHROPIC_API_KEY`
- `anthropic-version: 2023-06-01`

## Request Parameters

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `model` | string | ✅ | 模型 ID，如 `claude-opus-4-8`、`claude-sonnet-4-6` |
| `messages` | array | ✅ | 对话消息列表 |
| `max_tokens` | integer | ✅ | 最大生成 token 数 |
| `system` | string/array | 否 | 系统提示词（顶层，不在 messages 内） |
| `stream` | boolean | 否 | 流式响应，默认 false |
| `temperature` | number | 否 | 采样温度 0~1（Opus 4.7+ 不支持） |
| `top_p` | number | 否 | 核采样（Opus 4.7+ 不支持） |
| `top_k` | integer | 否 | Top-K 采样（Opus 4.7+ 不支持） |
| `stop_sequences` | array | 否 | 停止序列 |
| `tools` | array | 否 | 工具定义 |
| `tool_choice` | object | 否 | 工具选择策略 |
| `thinking` | object | 否 | 扩展思考配置：`{ "type": "adaptive" }` 或 `{ "type": "enabled", "budget_tokens": N }` |
| `metadata` | object | 否 | 自定义元数据 |

## Message Roles

| Role | 说明 |
|------|------|
| `user` | 用户消息 |
| `assistant` | 模型响应（用于多轮对话历史） |

## Content Block Types

| Type | 说明 |
|------|------|
| `text` | 文本内容 |
| `tool_use` | 工具调用请求 |
| `tool_result` | 工具调用结果（user 消息中） |
| `image` | 图片内容 |
| `thinking` | 思考过程（extended thinking 时返回） |
| `redacted_thinking` | 被截断的思考内容 |

## Stop Reasons

| 值 | 说明 |
|------|------|
| `end_turn` | 模型自然结束 |
| `max_tokens` | 达到 max_tokens 限制 |
| `stop_sequence` | 触发了自定义停止序列 |
| `tool_use` | 模型请求工具调用 |
| `pause_turn` | Extended thinking 暂停（Files API / computer use） |
| `refusal` | 内容安全拒绝 |

## Response Object

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 消息唯一 ID，如 `msg_01XFDUDYJgAACzvnptvVoYEL` |
| `type` | string | 固定值 `"message"` |
| `role` | string | 固定值 `"assistant"` |
| `content` | array | content block 数组 |
| `model` | string | 实际使用的模型 ID |
| `stop_reason` | string | 停止原因（见上表） |
| `stop_sequence` | string\|null | 触发的停止序列 |
| `usage` | object | token 用量：`input_tokens`、`output_tokens`、`cache_creation_input_tokens`、`cache_read_input_tokens` |

## 示例 1：基础 Message

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 16000,
    "messages": [
      {"role": "user", "content": "What is the capital of France?"}
    ]
  }'
```

### Response
```json
{
  "id": "msg_01XFDUDYJgAACzvnptvVoYEL",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "text",
      "text": "The capital of France is Paris."
    }
  ],
  "model": "claude-opus-4-8-20250514",
  "stop_reason": "end_turn",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 12,
    "output_tokens": 10
  }
}
```

## 示例 2：System Prompt

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 16000,
    "system": "You are a helpful assistant who speaks like a pirate.",
    "messages": [
      {"role": "user", "content": "Hello"}
    ]
  }'
```

**关键差异**：`system` 是顶层字段，不在 `messages` 数组内。与 OpenAI 的 `{"role": "system", "content": "..."}` 在 messages 内的做法不同。

## 示例 3：Multi-Turn Conversation

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "Hello, Claude"},
      {"role": "assistant", "content": "Hello! How can I help you?"},
      {"role": "user", "content": "Can you describe LLMs to me?"}
    ]
  }'
```

## 示例 4：Streaming

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 64000,
    "stream": true,
    "messages": [
      {"role": "user", "content": "Write a haiku"}
    ]
  }'
```

### Streaming Events（SSE 格式）

```
event: message_start
data: {"type":"message_start","message":{"id":"msg_xxx","type":"message","role":"assistant","content":[],"model":"claude-opus-4-8","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Calm"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" morning"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":15}}

event: message_stop
data: {"type":"message_stop"}
```

Streaming 事件类型：
| Event | 说明 |
|------|------|
| `message_start` | 消息开始，包含 message 元信息（id、model、usage.input_tokens） |
| `content_block_start` | content block 开始（text/tool_use 等） |
| `content_block_delta` | content block 增量内容（text_delta / input_json_delta） |
| `content_block_stop` | content block 结束 |
| `message_delta` | 消息级增量（stop_reason、usage.output_tokens） |
| `message_stop` | 消息流结束 |

## 示例 5：Tool Use

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 16000,
    "tools": [{
      "name": "get_weather",
      "description": "Get the current weather in a given location",
      "input_schema": {
        "type": "object",
        "properties": {
          "location": {
            "type": "string",
            "description": "The city and state, e.g. San Francisco, CA"
          }
        },
        "required": ["location"]
      }
    }],
    "messages": [
      {"role": "user", "content": "What is the weather like in Paris?"}
    ]
  }'
```

### Response
```json
{
  "id": "msg_01Aq9w938a90dw8q",
  "type": "message",
  "role": "assistant",
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_01D7FLrfh4GYq7yT1ULFeyMV",
      "name": "get_weather",
      "input": {
        "location": "Paris, France"
      }
    }
  ],
  "model": "claude-opus-4-8-20250514",
  "stop_reason": "tool_use",
  "stop_sequence": null,
  "usage": {
    "input_tokens": 50,
    "output_tokens": 20
  }
}
```

**关键差异 vs OpenAI**：
- Anthropic tools: `name` + `input_schema`（JSON Schema 格式）
- OpenAI tools: `type: "function"` + `function.name` + `function.parameters`
- Anthropic tool_use content block: `type: "tool_use"`, `id`, `name`, `input`（已解析的 JSON object）
- OpenAI tool_calls: `type: "function"`, `id`, `function.name`, `function.arguments`（JSON 字符串）

## 示例 6：Vision (Image)

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 1024,
    "messages": [{
      "role": "user",
      "content": [
        {
          "type": "image",
          "source": {
            "type": "url",
            "url": "https://example.com/image.jpg"
          }
        },
        {"type": "text", "text": "What is in this image?"}
      ]
    }]
  }'
```

Image source types:
- `url` — 图片 URL
- `base64` — Base64 编码的图片数据，需指定 `media_type` 和 `data`

Supported formats: `image/jpeg`, `image/png`, `image/gif`, `image/webp`

## 示例 7：Extended Thinking

### Request
```bash
curl https://api.anthropic.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 16000,
    "thinking": {
      "type": "adaptive"
    },
    "messages": [
      {"role": "user", "content": "Solve this complex math problem..."}
    ]
  }'
```

Thinking 配置：
- `{"type": "adaptive"}` — 模型自适应思考深度（Opus 4.6+ 推荐）
- `{"type": "enabled", "budget_tokens": 4000}` — 固定思考 token 预算

## Error Response 格式

```json
{
  "type": "error",
  "error": {
    "type": "invalid_request_error",
    "message": "Invalid model specified"
  }
}
```

常见 error.type：
- `invalid_request_error` — 请求参数错误
- `authentication_error` — API key 无效
- `permission_error` — 无权限访问
- `not_found_error` — 资源不存在
- `rate_limit_error` — 限流
- `api_error` — 服务端错误
- `overloaded_error` — 服务过载
