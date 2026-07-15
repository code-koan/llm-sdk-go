---
description: .claude package — AI 编程基础设施中央索引
---

# AI 基础设施索引

> 本项目所有 AI 编程基础设施的统一入口。AI 通过此文件发现、使用、维护自身工具链。

## Skills（开发/审计/沉淀）

| Skill | 触发 | 说明 |
|-------|------|------|
| [deep-coding](skills/deep-coding/SKILL.md) | 需求实现前后 | 唯一开发入口：L0 环境检查 → L1 方案设计 → L2 并发 agent → 合入 gate |
| [provider-adpter](skills/provider-adpter/SKILL.md) | 新增/大改 Provider | Provider 适配完整工作流 |
| [sdk-architecture](skills/sdk-architecture/SKILL.md) | 新增包/Provider 前、架构 review | 定义「合法代码库」边界 |
| [index-md](skills/index-md/SKILL.md) | 需求实现前检索、实现后沉淀 | `_index.md` 反向引用体系 |
| [reflecting](skills/reflecting/SKILL.md) | 会话收尾/review出问题/agent异常/手动 | 六层反思：情景学习→自我进化→经验捕获→文档沉淀→基础设施审视→过程审计 |
| [graphify](skills/graphify/SKILL.md) | 代码检索、理解 | 知识图谱查询/更新，代码检索首选 |
| [speckit-*](skills/speckit-specify/SKILL.md) | 新需求 | specify→clarify→plan→tasks→implement→converge 全链 |

## Agents（子代理）

**选型决策表 — 派发前必查**：

| 任务特征 | Agent | 判别关键词 |
|---------|-------|-----------|
| 机械替换、批量重命名、格式统一 | **lead sed/Edit**（不派 agent） | `s/X/Y/g`、纯文本替换 |
| 需要理解 Provider 规范、写功能代码 | [project-implementer](agents/project-implementer.md) | 新功能、Provider 适配、逻辑变更 |
| 定位符号、理解数据流、评估改动半径 | [Explore](agents/Explore.md) | 在哪定义、谁调用、影响面 |
| 代码审查、验收 | [project-reviewer](agents/project-reviewer.md) | PR review、stage 验收、架构审计 |

**判别标准**：任务可以用一句话完整描述且无歧义 → 低认知负荷，lead 脚本。需要「理解代码再决定怎么改」→ 高认知负荷，派 agent。

Agent 面对大量重复 Edit 会走捷径（正则/批处理）→ 产生破损输出。已验证：机械替换 ≥3 处，lead sed 链一次过，不要派 agent。

## Output Styles

| Style | 效果 |
|-------|------|
| [project-ai-style](output-styles/project-ai-style.md) | 高信息密度、结构化、不写叙述 |

## 自审规则

AI 通过 `reflecting` 维度三维护自身基础设施。每次自审覆盖：

```
□ 去重        — skills/agents 间无重复规则
□ 过期        — 所有引用路径有效
□ 触发条件    — 每个 skill 的触发条件准确
□ _index.md   — 覆盖全部 skills/agents/output-styles
□ memory      — 已验证的反馈已沉淀为 memory 文件
```
