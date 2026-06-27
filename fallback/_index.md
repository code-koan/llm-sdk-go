---
description: Router 多 Provider 负载分发、选择和重试策略实现
---

# fallback

Router 实现，提供多 Provider 负载分发、选择和重试策略。

## 文件

| 文件 | 职责 | 设计文档 |
|------|------|----------|
| `fallback.go` | Router 主实现，实现 Provider + CapabilityProvider 接口 | [Fallback 文档](../docs/fallback.md) |
| `retry.go` | 重试策略定义与内置实现 | [Fallback 文档](../docs/fallback.md) |
| `selectors.go` | Provider 选择策略（Selector 接口及内置实现） | [Fallback 文档](../docs/fallback.md) |

## 设计文档

→ [架构文档](../docs/architecture.md)
→ [Fallback 文档](../docs/fallback.md)
