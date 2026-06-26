---
name: deep-coding
description: >
  这是 llm-sdk-go 的深度编码规范，任何涉及统一接口、错误标准化、流式响应、配置、测试或文档的开发都必须先按此流程规划和验收。
  Provider 开发请走 /provider-adpter。
  触发时机：
  1. 需求实现前——判断是否涉及 SDK 公共接口、OpenAI 兼容格式、流式响应或错误归一化
  2. 需求实现后——更新测试与 docs，确保根包 re-export 和文档保持同步
  3. 手动触发——/deep-coding
---

# llm-sdk-go 深度编码规范【IMPORTANT】

## 第一性原理

llm-sdk-go 的核心目标只有三个：

1. **统一接口**：调用方用同一套 `providers.Provider` 接口访问不同 LLM 厂商。
2. **OpenAI 兼容输出**：响应结构尽量标准化为 OpenAI 兼容格式，减少上层适配成本。
3. **Go SDK 可维护性**：零不必要抽象、低依赖、易测试、行为稳定。

任何实现都从这三个目标反推：如果改动不能提升接口一致性、兼容输出或可维护性，就不要做。

## 代码开发规范

- 接口优先：Provider 必须实现 `providers.Provider`，可选能力通过 `CapabilityProvider`、`EmbeddingProvider`、`ModelLister`、`ErrorConverter` 等接口表达。
- 不破坏公共 API：修改 `providers/types.go`、根包 `llmsdk.go` 或 exported 类型前，必须评估调用方兼容性。
- Provider 行为一致：新增/修改 Provider 时，对 Completion、CompletionStream、错误转换、配置校验、日志字段保持与参考实现一致。
- OpenAI 兼容 Provider 优先复用 `providers/openai/compatible.go`，不要复制 OpenAI 兼容协议代码。
- 配置使用 functional options，默认值明确，必填项在发起 API 请求前验证。
- 错误归一化使用 SDK typed errors + `errors.As`，避免字符串匹配；无法 typed 判断时要说明原因。
- 流式响应必须处理 `ctx.Done()`，向 channel 发送时使用 `select`，避免调用方停止消费后 goroutine 永久阻塞。
- 不修改 receiver state 或入参；需要保留切片时使用 `slices.Clone`。
- 魔法字符串必须抽常量，常量放生产代码，不只放测试。
- 结构体字段按 A-Z 排序；函数短小、早返回、避免深层嵌套。
- 测试禁止真实网络请求；集成测试无凭据或服务不可用时 graceful skip。

## 开发流程（CRITICAL — 严格执行顺序）

### Step 0：环境检查

```
□ git status --short          → 确认已有改动归属，避免覆盖用户工作
□ git branch --show-current   → 确认当前分支与 issue / 任务一致
□ make build 或 go test ./...  → 必要时确认基线是否可用
```

### Step 1：需求确认关卡

写任何代码前：

- 阅读关联 issue / PR / 用户描述，提取所有验收点。
- 判断是否影响公共接口、Provider 行为、错误语义、流式协议、文档示例。
- 调研参考实现，优先对齐 `providers/anthropic/` 和 `providers/openai/` 的现有模式。
- 输出简洁开发计划；若是多文件或公共 API 变更，必须等用户明确确认后再写代码。

### Step 2：Provider 设计关卡 → 走 `/provider-adpter`

### Step 3：实现顺序 → Provider 相关走 `/provider-adpter`，通用代码按依赖从底到上推进

### Step 4：Issue/PR 留痕

- 开发计划确认后，如有关联 issue，使用 `gh issue comment <issue> --body "..."` 留下计划摘要。
- PR body 必须关联 issue，说明公共 API 影响、Provider 兼容策略、测试覆盖和无法本地验证的外部依赖。

## Provider 实现

→ 见 `/provider-adpter` Phase 3 检查表

## 测试规范

- 使用 `require`，不用 `assert`。
- table-driven case 变量命名为 `tc`，不用 `tt`。
- 除 `t.Setenv()` 场景外优先 `t.Parallel()`。
- helper 必须 `t.Helper()`。
- mock/fake/test 命名明确，不与生产实现混淆。
- 测试中不发真实网络请求：用 `httptest.Server`、fake client、round tripper 或跳过集成测试。
- 基础包要有自己的测试，不只依赖 wrapper 测试间接覆盖。
- 避免重复断言：`require.NotEmpty` 后不要再检查 `len > 0`。
- 故意丢弃返回值时写注释或使用合适的 nolint。

### Provider 测试

→ 见 `/provider-adpter` Phase 4

## 文档闭环

实现完成后检查：

```
□ README 或示例             → 公共用法变更已同步（如有）
□ llmsdk.go                 → 需要根包 re-export 时已同步
□ CHANGELOG.md              → 用户可见行为变更已记录（如项目当前维护该条目）
□ docs/providers.md         → Provider 列表和能力矩阵 → 走 `/provider-adpter` Phase 5
```

如果新增反复出现的开发约束，优先沉淀到 `CLAUDE.md` 或本 skill，避免只停留在对话中。

## 开发闭环检查表（按顺序执行）

```
□ gofmt -w <changed_go_files>
□ make build
□ make test-only
□ make lint 或 make test
□ 文档更新已完成
□ git diff --check
□ PR / issue 留痕已完成（如适用）
```

如果某一步因外部凭据、网络或环境不可用跳过，必须在最终回复和 PR body 中明确说明。

## AI coding 技巧

### 上下文管理

- 先读 `CLAUDE.md`、目标 Provider、参考 Provider，不要全仓库无差别读取。
- 先列验收标准，再决定读哪些文件；避免为简单修复引入过长上下文。
- 多文件变更先拆分：类型/配置、转换、stream、测试、文档，各组文件边界清晰。

### Subagent 调度

适合并行的任务：

```
Wave 0 (串行): 主流程确认公共接口、Provider 边界、测试策略
     ↓
Wave 1 (并行): conversion/helpers、streaming、error mapping、docs/tests 独立文件组
     ↓
主流程聚合: 读取关键实现，确认接口一致性和测试意图
     ↓
Wave 2 (串行): gofmt + build + test + lint + 文档核对
```

- 同一工作树并发时必须限制互不重叠文件；会改相同 Provider 文件时不要并行。
- subagent 提示必须写清：必须写代码，不只输出 plan；只改指定文件；不扩大重构范围。
- subagent 自报通过不能直接相信；主流程必须重新读取关键文件并统一跑验证命令。
- 验证失败先收敛失败面，不继续扩大并发。

## PR Review 检查表

```
□ 公共接口兼容：exported 类型、方法签名、根包 re-export 无意外破坏
□ Provider 一致性：Completion / Stream / Error / Config 行为与参考实现对齐
□ 流式安全：ctx cancel、channel close、错误传播无 goroutine 泄漏
□ 错误归一：typed errors + errors.As，sentinel errors 可被 errors.Is 命中
□ OpenAI 兼容：响应字段、finish reason、usage、role/content 转换合理
□ 配置安全：不泄露密钥，env var 和默认值文档一致
□ 测试隔离：无真实网络请求，集成测试可跳过
□ 简洁性：没有无必要抽象、全局可变状态、复制粘贴的大块协议代码
□ 文档完整：Provider 文档、示例、能力边界已同步
```
