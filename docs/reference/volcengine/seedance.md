# Seedance 视频生成 API

> 来源：https://www.volcengine.com/docs/82379/1520757
> 存档日期：2025-06-26

## Endpoint

### 创建任务
POST https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks
Authorization: Bearer $ARK_API_KEY

### 查询任务
GET https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/{id}

## 模型列表

| 模型 ID | 系列 | 能力 | 分辨率 | 音频 |
|---------|------|------|--------|------|
| doubao-seedance-2-0-260128 | 2.0 标准 | 文/图/视频/音频生视频 | 最高 1080p | ✅ |
| doubao-seedance-2-0-fast-260128 | 2.0 Fast | 文/图/视频/音频生视频 | 最高 720p | ✅ |
| doubao-seedance-1-5-pro-251215 | 1.5 Pro | 文/图生视频 | 720p | ✅ 音画同步 |
| doubao-seedance-1-0-pro-250528 | 1.0 Pro | 文/图生视频（首尾帧） | 1080p | ❌ |
| doubao-seedance-1-0-pro-fast-251015 | 1.0 Pro Fast | 文/图生视频 | 720p | ❌ |

## Request Parameters

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| model | string | ✅ | 模型 ID |
| content | array | ✅ | 输入内容（text/image_url/audio_url/video_url） |
| resolution | string | ❌ | 480p / 720p / 1080p |
| ratio | string | ❌ | 16:9 / 4:3 / 1:1 / 3:4 / 9:16 / 21:9 / adaptive |
| duration | int | ❌ | 2.0: 4-15s，1.x: 2-12s |
| generate_audio | bool | ❌ | 是否生成同步音频（1.5 Pro/2.0） |
| watermark | bool | ❌ | 是否添加水印 |
| seed | int | ❌ | 随机种子（2.0 不支持） |
| return_last_frame | bool | ❌ | 是否返回尾帧图像 |
| callback_url | string | ❌ | 任务状态回调地址 |
| service_tier | string | ❌ | default(在线) / flex(离线，半价) |

## Content 类型

### text
```json
{ "type": "text", "text": "写实风格，晴朗的蓝天之下，一大片白色的雏菊花田，镜头逐渐拉近" }
```

### image_url（首帧）
```json
{ "type": "image_url", "image_url": { "url": "https://example.com/first_frame.png" }, "role": "first_frame" }
```

### image_url（尾帧）
```json
{ "type": "image_url", "image_url": { "url": "https://example.com/last_frame.png" }, "role": "last_frame" }
```

### image_url（参考图，2.0 支持 0-9 张）
```json
{ "type": "image_url", "image_url": { "url": "https://example.com/ref.png" }, "role": "reference" }
```

### video_url（参考视频，2.0 支持 0-3 个）
```json
{ "type": "video_url", "video_url": { "url": "https://example.com/ref.mp4" } }
```

### audio_url（参考音频，2.0 支持 0-3 个）
```json
{ "type": "audio_url", "audio_url": { "url": "https://example.com/ref.mp3" } }
```

## 生成模式 (content 组合)

| 模式 | content 组合 | 支持版本 |
|------|-------------|---------|
| 文生视频 | 1 个 text | 全部 |
| 图生视频-首帧 | text(可选) + 1 个 image_url(role: first_frame) | 全部 |
| 图生视频-首尾帧 | text(可选) + 2 个 image_url(first_frame + last_frame) | 1.0 Pro/1.5 Pro/2.0 |
| 多模态参考 | text + 0-9 参考图 + 0-3 参考视频 + 0-3 参考音频 | 仅 2.0 |

## 完整示例：文生视频

### Request
```bash
curl --location "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks" \
  --header "Content-Type: application/json" \
  --header "Authorization: Bearer $ARK_API_KEY" \
  --data '{
    "model": "doubao-seedance-2-0-260128",
    "content": [
      {
        "type": "text",
        "text": "写实风格，晴朗的蓝天之下，一大片白色的雏菊花田，镜头逐渐拉近，花瓣上有几颗晶莹的露珠"
      }
    ],
    "ratio": "16:9",
    "duration": 5,
    "resolution": "720p",
    "watermark": false
  }'
```

### 创建 Response
```json
{ "id": "cgt-20251125163544-qrj4f" }
```

### 查询 Response（succeeded）
```json
{
  "id": "cgt-20251119202422-jcfm2",
  "model": "doubao-seedance-2-0-260128",
  "status": "succeeded",
  "content": {
    "video_url": "https://ark-content-generation-cn-beijing.tos-cn-beijing.volces.com/xxx.mp4"
  },
  "usage": { "completion_tokens": 295800 },
  "duration": 5,
  "resolution": "1080p",
  "ratio": "9:16"
}
```

### 查询 Response（failed）
```json
{
  "id": "cgt-20251119202422-jcfm2",
  "status": "failed",
  "error": {
    "code": "InvalidParameter",
    "message": "Invalid parameter: duration"
  }
}
```

## 完整示例：图生视频（首帧）

### Request
```bash
curl --location "https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks" \
  --header "Content-Type: application/json" \
  --header "Authorization: Bearer $ARK_API_KEY" \
  --data '{
    "model": "doubao-seedance-2-0-260128",
    "content": [
      {
        "type": "image_url",
        "image_url": { "url": "https://example.com/first_frame.png" },
        "role": "first_frame"
      },
      {
        "type": "text",
        "text": "让画面中的花朵随风摇曳"
      }
    ],
    "duration": 5
  }'
```

## 任务状态

| status | 说明 |
|--------|------|
| queued | 排队中 |
| running | 生成中 |
| succeeded | 成功，video_url 24h 后过期 |
| failed | 失败，查看 error 字段 |
| expired | 超时 |

## Error 格式

```json
{
  "error": {
    "code": "InvalidParameter",
    "message": "Invalid parameter: duration",
    "param": "duration"
  }
}
```

常见 error.code：
- InvalidParameter — 参数错误
- AuthenticationError — API key 无效
- RateLimitExceeded — 限流
- InternalError — 服务内部错误
- ResourceNotFound — 任务/模型不存在
