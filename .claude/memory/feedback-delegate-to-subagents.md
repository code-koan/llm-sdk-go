---
name: feedback-delegate-to-subagents
description: Split work into subagent tasks for multi-file changes — when and when NOT to delegate
metadata:
  type: feedback
---

# Delegate to Subagents — Decision Rule

## Rule

| 改动文件数 | 执行方式 |
|-----------|---------|
| <5 文件，机械替换 | **lead 直接 sed/Edit** |
| <5 文件，需理解代码 | lead 直接改（上下文可控） |
| 5+ 文件，需理解代码 | 派 agent（按文件边界拆分） |
| 任意文件数，纯 `s/X/Y/g` | **永远不派 agent** — lead sed 链 |

Agent 面对大量重复 Edit 会走捷径 → 产生破损输出。已验证：20 文件机械重命名，4 agent 全被 kill，lead sed 30 秒修完。

## Pre-dispatch checklist

```
□ 这是机械替换吗？（>50% 改动是 rename/delete）→ lead sed，不派
□ 需要 project-implementer 吗？（需理解 Provider 规范）→ 派 project-implementer
□ prompt 第一行写了 IMPLEMENTATION TASK (not planning) 吗？→ 必须写
□ aget 类型选对了吗？→ 查 _index.md agent 选型表
```

**Why:** 本次会话（2026-07-15）4 个 general-purpose agent 全被 kill，原因是机械重命名不该派 agent。

**How to apply:** 派 agent 前过 checklist。

See also: [[subagent-implementation-not-planning]], [[subagent-output-verification]]
