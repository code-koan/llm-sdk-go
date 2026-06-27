# Seedance Video Generation

> import: `"github.com/code-koan/llm-sdk-go/providers/volcengine/seedance"`

Seedance 是字节跳动豆包团队的视频生成模型，支持文生视频、图生视频（首帧/首尾帧）、多模态参考生视频。

## 环境变量
| 变量 | 必需 | 说明 |
|------|------|------|
| `ARK_API_KEY` | ✅ | 火山方舟 API Key |

## 快速上手
### 文生视频
```go
p, _ := seedance.New()
task, _ := p.SubmitTask(ctx, providers.AsyncTaskParams{
    Model:   "doubao-seedance-2-0-260128",
    Content: "写实风格，晴朗的蓝天之下，一大片白色的雏菊花田，镜头逐渐拉近",
    Extra: map[string]any{
        "resolution": "720p",
        "ratio":      "16:9",
        "duration":   5,
    },
})
for {
    task, _ = p.GetTask(ctx, task.ID)
    if task.Status == providers.AsyncTaskSucceeded {
        fmt.Println(task.ResultURL)
        break
    }
    if task.Status == providers.AsyncTaskFailed {
        fmt.Printf("failed: %s - %s\n", task.Error.Code, task.Error.Message)
        break
    }
    time.Sleep(2 * time.Second)
}
```
### 图生视频（首帧）
```go
task, _ := p.SubmitTask(ctx, providers.AsyncTaskParams{
    Model: "doubao-seedance-2-0-260128",
    Content: []providers.ContentPart{
        {Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/first_frame.jpg", Role: "first_frame"}},
        {Type: "text", Text: "让画面中的花朵随风摇曳"},
    },
    Extra: map[string]any{"duration": 5},
})
```
### 图生视频（首尾帧）
```go
task, _ := p.SubmitTask(ctx, providers.AsyncTaskParams{
    Model: "doubao-seedance-1-0-pro-250528",
    Content: []providers.ContentPart{
        {Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/first.png", Role: "first_frame"}},
        {Type: "image_url", ImageURL: &providers.ImageURL{URL: "https://example.com/last.png", Role: "last_frame"}},
        {Type: "text", Text: "从首帧过渡到尾帧"},
    },
})
```
## 模型列表
| 模型 ID | 分辨率 | 音频 | 首尾帧 | 多模态参考 |
|---------|--------|------|--------|-----------|
| doubao-seedance-2-0-260128 | 最高 1080p | ✅ | ✅ | ✅ (图/视频/音频) |
| doubao-seedance-2-0-fast-260128 | 最高 720p | ✅ | ✅ | ✅ |
| doubao-seedance-1-5-pro-251215 | 720p | ✅ 音画同步 | ✅ | ❌ |
| doubao-seedance-1-0-pro-250528 | 1080p | ❌ | ✅ | ❌ |
| doubao-seedance-1-0-pro-fast-251015 | 720p | ❌ | ✅ | ❌ |

## Extra 参数
| 参数 | 类型 | 说明 |
|------|------|------|
| resolution | string | 480p / 720p / 1080p |
| ratio | string | 16:9 / 9:16 / 4:3 / 1:1 / adaptive |
| duration | int | 2.0: 4-15s，1.x: 2-12s |
| generate_audio | bool | 生成同步音频（1.5 Pro/2.0） |
| watermark | bool | 是否添加水印 |
| seed | int | 随机种子（2.0 不支持） |
| return_last_frame | bool | 是否返回尾帧图像 |
| callback_url | string | 任务完成回调地址 |
| service_tier | string | default(在线) / flex(离线，半价) |

## 能力矩阵
| 能力 | 支持 |
|------|------|
| AsyncGeneration | ✅ |
| Completion | ❌ |
| CompletionStreaming | ❌ |

## 配置选项
```go
p, _ := seedance.New(
    config.WithAPIKey("your-api-key"),
    config.WithBaseURL("https://..."),
    config.WithLogger(myLogger),
    config.WithHTTPClient(myClient),
)
```

## 接口
实现了 `providers.AsyncTaskProvider`、`providers.Provider`、`providers.CapabilityProvider`、`providers.ErrorConverter`。`SubmitTask` 发起视频生成任务，`GetTask` 查询任务状态/结果（视频 URL 24h 后过期）。`Completion` / `CompletionStream` 不支持，调用返回错误。
