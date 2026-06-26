---
name: provider-adpter
description: >
  llm-sdk-go 新增和适配 Provider 的完整开发工作流。
  触发时机：
  1. 新增 Provider / 大改现有 Provider 时自动激活
  2. 手动触发——/provider-adpter
---

# llm-sdk-go Provider 集成规范

## 第一性原理

Provider 集成的核心目标：
1. 实现 providers.Provider 统一接口
2. 响应归一化为 OpenAI 兼容格式
3. 错误可被 errors 包 sentinel errors 精准命中

## 开发闭环（严格顺序执行）

### Phase 0：官方文档存档

**目的**：将官方文档中的请求/响应示例固化为客观真值，作为 Phase 2 TDD 测试期望的唯一来源。

**存档内容**（存入 `docs/reference/{provider}/`）：
- API reference 页面（endpoint、参数、响应字段说明）
- **官方 curl 示例 + 完整 JSON 响应** ← TDD 关键输入
- 模型能力说明（支持的参数、finish reason、token 计算方式等）
- 错误码与 HTTP 状态码映射表

**约束**：
- 来源必须是官方文档（api.xxx.com/docs、platform.xxx.com/docs），不接受二手博客/论坛
- 保存为 markdown，JSON 示例必须完整、不经手工修改
- 存档后作为客观真值，TDD 测试期望严格对齐存档中的示例
- 使用 index-docs 维护 `docs/reference/{provider}/` 的 _index.md

### Phase 1：Provider 设计关卡

写代码前明确：
□ 厂商 API 类型：原生 SDK / OpenAI-compatible / HTTP wrapper
□ 必填配置：API key、base URL、默认 model、timeout
□ 支持能力：completion / streaming / embedding / model listing
□ 错误映射：认证、限流、上下文长度、内容过滤、模型不存在、无效请求 → SDK typed errors
□ 响应归一：ID、Object、Created、Model、Choices、Usage 是否可稳定填充
□ 流式语义：chunk 顺序、finish reason、usage 是否可获得

### Phase 2：TDD 实现（按依赖顺序）

**TDD 闭环**（step 4-6 严格执行，step 1-3 常规测试）：

```
Phase 0 存档的官方示例 → 写测试（红，失败）→ 最小实现（绿，通过）→ 重构（保持绿）
```

- 测试的期望输入/输出**直接取自 Phase 0 存档的官方 JSON 示例**，不自行编造
- 每个最小实现单元必须能追溯到 `docs/reference/{provider}/` 中的具体示例

实现顺序：
1. package constants / types / options → 常规测试，验证配置正确性
2. constructor + config validation → 常规测试，验证必填校验
3. request/response conversion helpers → 常规测试，验证结构转换
4. Completion → **严格 TDD**，取官方 curl 示例的 request 作为输入，官方 JSON response 作为期望输出
5. CompletionStream → **严格 TDD**，取官方 SSE 示例验证 chunk 顺序、finish reason、channel close
6. optional interfaces: Capabilities / Embedding / ListModels / ConvertError → **TDD**，有官方示例的以示例为期望
7. 文档 `docs/api/{provider}/` SDK 使用说明

文件接近 800 行或职责混杂时拆出 conversion、stream、errors、options 文件。

### Phase 3：Provider 实现检查表

□ package 名与目录名一致
□ interface assertions 完整：Provider / optional interfaces
□ New(...) 默认值和必填校验明确
□ Completion 请求前校验 Model 和 Messages
□ CompletionStream goroutine 响应 ctx.Done()
□ channel send 使用 select，消费方退出不阻塞 goroutine
□ Usage、FinishReason、Tool/JSON response format 等字段标准化
□ ConvertError 覆盖 SDK typed errors，映射到 errors 包 sentinel errors
□ 日志只记录调试信息，不泄露 API key / token / secret
□ 所有 magic strings 已抽常量
□ 不把 provider SDK 类型泄露到公共统一接口
□ OpenAI-compatible wrapper 显式覆盖 CompatibleConfig 所有字段

### Phase 4：Provider 测试覆盖最小集

□ New 默认配置和 option 覆盖
□ 缺少 API key / model / messages 的校验错误
□ Completion 成功路径：请求转换 + 响应转换 + usage
□ Completion 错误路径：typed error → normalized sentinel error
□ CompletionStream 成功路径：多个 chunk + finish reason + close channel
□ CompletionStream ctx cancel：不泄露 goroutine，不阻塞
□ Capabilities / Embedding / ListModels（如实现）
□ 每个最小测试有对应的官方文档示例作为期望值来源

### Phase 5：文档闭环

□ `docs/api/{provider}/` SDK 使用说明已创建/更新
□ `docs/providers.md` Provider 列表和能力矩阵已更新
□ `docs/_index.md` 已更新引用
□ 根包 `llmsdk.go` re-export（如需要）
□ CHANGELOG（如项目维护）

## 顶层抽象原则

- SDK 保持简洁抽象，顶层接口变更必须人工决策后实现
- 不确定的抽象不引入，先用具体实现验证再考虑泛化
- 优先复用 `providers/openai/compatible.go`，不复制 OpenAI 兼容协议代码

## 开发闭环验证

□ gofmt -w <changed_files>
□ make build
□ make test-only
□ make lint
□ 文档更新已完成
□ git diff --check
