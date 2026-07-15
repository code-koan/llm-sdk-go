---
description: Internal test utilities — FakeServer, fixtures, MockProvider for provider tests
---

# internal/testutil

Provider 测试的公共工具：HTTP fake server、测试固定值、Mock Provider。

## 核心文件

| 文件 | 职责 |
|------|------|
| `fakeserver.go` | httptest.Server + handler 注册，模拟 LLM API 响应 |
| `fixtures.go` | 测试固定值：示例请求/响应/配置 |
| `mocks.go` | MockProvider 实现，满足 Provider 接口 |
