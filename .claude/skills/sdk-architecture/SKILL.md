---
name: sdk-architecture
description: >
  llm-sdk-go 架构权威 — 定义「合法代码库」的样子。
  触发：
  1. 新增包/Provider 前 — 确认边界合规
  2. 架构 review / PR review — 对照检查
  3. 手动触发 — /sdk-architecture
---
# llm-sdk-go 领域架构

> 定义 WHAT（合法架构），不重复 HOW（见 [deep-coding](../deep-coding/SKILL.md)、[provider-adpter](../provider-adpter/SKILL.md)）。

## 第一性原理

1. **统一接口** — `providers.Provider` 是所有 LLM 访问的单一出入口。
2. **OpenAI 兼容输出** — 响应标准化为 OpenAI 格式，减少上层适配。
3. **零不必要抽象** — 一个接口一个实现不抽 interface，一个工厂一个产品不抽 factory。

从这三个目标反推：改动提升接口一致性/输出兼容性/简洁性则做，否则不做。

## 包架构与边界

| 包 | 职责 | 可导入 | 可被导入 |
|----|------|--------|---------|
| 根包 `llmsdk.go` | re-export 公共类型 | 仅 errors | 外部消费者 |
| `config/` | 配置 + Logger 接口 | 无内部依赖 | 任何包 |
| `errors/` | sentinel errors + 工厂函数 | 无内部依赖 | 任何包 |
| `providers/` | 接口定义 + 共享类型 | errors, config | 根包, fallback, callers |
| `providers/*/` | Provider 实现 | 仅 providers/types, config, errors | 不被其他包导入（仅通过接口访问） |
| `protocol/` | 厂商原生类型归一化桥梁 | 仅 providers | 仅 `internal/`, `cmd/` |
| `fallback/` | 重试/回退/选择器 | 仅 providers/types, config, errors | 外部消费者 |
| `param/` | 参数包装 | 仅 providers/types | 外部消费者 |
| `internal/` | 不可导出工具 | 无限制（但限 internal） | 仅同树包 |
| `examples/` | 使用示例 | 任意 | 不导入 |
| `cmd/` | CLI 工具 | 任意 | 不导入 |

### 跨边界铁律

- **禁止** `providers/*/` 互相导入 — Provider 间无关
- **禁止** `providers/*/` 导入 `protocol/` — protocol 是归一化桥梁，非 Provider 工具包
- **禁止** `providers/types.go` 导入任何 provider 实现 — 接口层对实现层零依赖
- **允许** `internal/testutil` 导入任意 `providers/*/` — 测试工具例外

## Provider 最小骨架

```
providers/{name}/
  {name}.go            # 接口断言 + 构造函数 + Completion + CompletionStream + 可选接口
  [{name}_test.go]     # 测试（必需）
  [errors.go]          # 错误转换（≥3 error type 时拆）
  [messages.go]        # 消息转换（≥50 行转换逻辑时拆）
  [stream.go]          # 流式处理（复杂流式逻辑时拆）
  [options.go]         # 配置选项（≥3 个 option 时拆）
```

- 单文件对简单 Provider 有效 — 800 行或职责混杂时拆
- **必须** 包含接口断言（`var _ Provider = (*Type)(nil)`）
- **禁止** `*_gen.go` / `*.gen.go` 纳入人工编辑范围
- **禁止** provider SDK 类型泄露到公共接口返回值

| 模式 | 参考文件 |
|------|---------|
| 原生 SDK Provider | `providers/anthropic/anthropic.go` |
| OpenAI 兼容 Provider | `providers/openai/compatible.go` + `groq/groq.go`（薄包装） |
| 最小单文件 Provider | `providers/ollama/ollama.go` |

## 接口层级

```
Provider (必需)
├── Name(), Completion(ctx, params), CompletionStream(ctx, params)
│
├── CapabilityProvider (可选) — Capabilities()
├── EmbeddingProvider (可选) — Embedding(ctx, params)
├── ModelLister (可选) — ListModels(ctx)
└── ErrorConverter (可选) — ConvertError(err) — 推荐所有 Provider 实现
```

- Optional interfaces 通过类型断言发现：`if ec, ok := p.(ErrorConverter); ok { ... }`

## 数据流

```
同步: Consumer → Completion(ctx, params) → 输入验证 → 请求转换 → API 调用
        └─ 成功 → 响应转换 → ChatCompletion
        └─ 失败 → ConvertError → sentinel error

流式: Consumer → CompletionStream(ctx, params) → 输入验证 → 请求转换 → 启动 goroutine
        ├─ for each chunk: select { case ch <- v: case <-ctx.Done() }
        ├─ final chunk (含累加 Usage) → close(ch)
        └─ 返回 <-chan ChatCompletionChunk

日志: 请求前 Debug(request params), 完成后 Debug(response/error/stream usage)
```

**关键设计点**:
- `select` + `ctx.Done()` 是所有 goroutine→channel 发送的硬要求
- 流式 goroutine 内部独立处理 cancel/超时，不依赖消费者主动 close
- Usage 在流结束时通过 final chunk 传递

## 横切规则（架构不变量）

| 规则 | 违规后果 |
|------|---------|
| `providers/types.go` 接口变更必须验证所有 Provider 编译通过 | 编译断裂 |
| `errors/errors.go` 新增 sentinel error 必须同步 ConvertError | 错误无法被命中 |
| goroutine→channel send 必须 `select { case ch <- v: case <-ctx.Done(): }` | goroutine 泄漏 |
| 传入切片必须 `slices.Clone` 后使用 | 调用方数据损坏 |
| `ContentString()` 代替 `msg.Content.(string)` | 非 string Content 引发 panic |
| 魔法字符串必须在生产代码抽常量（非测试文件） | 测试与实现解耦 |

## 三技能关系

```
sdk-architecture (WHAT — 架构权威)
  │ 定义合法边界、骨架、不变量
  ▼
deep-coding (HOW — 通用开发工作流)  →  PR Review 检查表引用横切规则
  │ 参考骨架做实现，遵守不变量
  ▼
provider-adpter (HOW-Provider — Provider 专用工作流)
     Phase 3/4 检查表由骨架派生，对齐 sdk-architecture
```

## Review 维度

| 维度 | 检查项 |
|------|--------|
| 接口合规 | Provider 全部方法实现、接口断言存在、可选接口按需实现 |
| 边界合规 | `providers/*/` 不互相导入、不导入 protocol、types 不依赖实现 |
| 骨架完整 | 文件结构符合最小骨架、多文件拆分合理、800 行触发拆文件 |
| 横切规则 | 流式安全、输入验证、错误转换、slice clone、ContentString、魔法字符常量 |
| Ponytail | 无多余抽象、无未用接口/方法/常量、单文件无过度拆分 |
