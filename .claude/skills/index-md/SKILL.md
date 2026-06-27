---
name: index-md
description: >
  通过 _index.md 反向引用体系管理设计文档和代码的追溯关系。
  触发时机：
  1. 需求实现前——检索 _index.md 找到相关设计文档，理解设计意图
  2. 需求实现后——沉淀文档：创建/更新 _index.md，记录代码→设计文档的引用
  3. 手动触发——/index-docs [search|update|check]
  当用户提到 "_index.md"、"反向引用"、"设计文档"、"文档沉淀"、"接口文档在哪"、"这个代码对应的设计文档" 时自动激活。
argument-hint: "[search|update|check <目录路径>]"
---

# _index.md 反向引用管理

**核心原则**: `_index.md` 实现 **代码 → 设计文档** 的双向追溯。
- 设计文档在 `docs/`，代码目录 `_index.md` 反向引用 `docs/`
- `docs/_index.md` 正向索引所有设计文档
- 闭环：`docs/_index.md` → 代码目录 `_index.md` → `docs/`

## 工作流程

### 模式一：检索（需求实现前）

1. 读目标目录的 `_index.md`，找到关联设计文档
2. 读 `docs/` 下的设计文档，理解设计意图
3. 汇报约束后给出实现建议

### 模式二：沉淀（需求实现后）

1. 确认变更涉及哪些目录
2. 检查/创建 `_index.md`
3. 新增设计文档时更新 `docs/_index.md`
4. 更新代码目录的 `_index.md` 引用

**_index.md 模板**：
```markdown
---
description: 目录职责一句话
---
# 目录名
## 文件
| 文件 | 职责 | 设计文档 |
|------|------|----------|
| `file.go` | 职责 | [设计文档](../docs/xxx.md) |
## 设计文档
→ [架构文档](../docs/architecture.md)
```

### 模式三：检查（手动触发）

检查完整性：列出 `docs/` 文档 → 列出源码目录 → 检查 `_index.md` 缺失 → 验证 `docs/_index.md` 全覆盖 → 验证双向引用一致 → 报告缺失。

```bash
# 列出所有 _index.md（docs/_index.md 正常出现）
find . -name "_index.md" -not -path "./.claude/*"
# 检查有 3+ .go 文件却无 _index.md 的目录
find . -type d -not -path "./.claude/*" -not -path "./.git/*" \
  -not -path "./docs/*" -not -path "./.codegraph/*" \
  -exec sh -c 'n=$(find "$1" -maxdepth 1 -name "*.go" | wc -l)
  [ "$n" -ge 3 ] && [ ! -f "$1/_index.md" ] && echo "缺失: $1 ($n files)"' _ {} \;
```

## 设计文档目录结构

```
docs/
├── _index.md                   ← 总索引（必需）
├── architecture.md             ← 架构设计
├── quickstart.md / providers.md / fallback.md
├── api/                        ← API 文档
│   ├── completion.md / streaming.md / errors.md / cache-and-ratelimit.md
│   └── {provider}/             ← 各 Provider SDK 使用说明（如 openai/、anthropic/）
│       └── ...
└── reference/                  ← 外部引用来源
    └── {provider}/             ← 各 Provider 官方文档存档
        └── ...
```

> 文件少于 3 个的源码目录可暂不创建 `_index.md`。

## 路径计算

从源码目录引用 `docs/` 使用相对路径。例：`providers/_index.md` → `docs/architecture.md` = `../docs/architecture.md`

## 注意事项

- 只记引用，不重复代码文档
- 设计文档存在才引用
- 新增设计文档时同步更新 `docs/_index.md`
- 目录变更时修正 `_index.md` 相对路径
- 引用 `docs/reference/` 的外部资料时标记出处
- Provider 文档按 provider-adpter 工作流维护：`docs/reference/{provider}/` 由 Phase 0 创建，`docs/api/{provider}/` 由 Phase 2/5 创建；以上目录的 `_index.md` 均纳入本体系
