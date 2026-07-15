---
name: index-md
description: >
  通过 _index.md 反向引用体系管理设计文档和代码的追溯关系。
  触发：需求实现前检索、实现后沉淀、手动 /index-md [search|update|check]
argument-hint: "[search|update|check <目录路径>]"
---
# _index.md 反向引用管理

> 系统架构以 [sdk-architecture](../sdk-architecture/SKILL.md) 为权威标准。

## 文档结构

```
docs/
  _index.md                   ← 总索引，列出所有文档
  architecture.md             ← 整体架构：接口体系 + Provider 规范 + 数据流
  providers.md                ← Provider 列表 + 能力矩阵
  quickstart.md               ← 最小可用示例
  fallback.md                 ← Router 负载分发与容灾
  model-capabilities.md       ← ChatModel + ChatBuilder 用法
  tokenizer.md                ← Token 估算 API
  tools.md                    ← Tool 代码生成
  api/                        ← SDK 使用文档
    {provider}/               ←   各 Provider 使用说明（含代码示例）
  reference/                  ← 官方 API 参考存档
    {provider}/               ←   各 Provider 官方文档（TDD 客观真值来源）
```

## 四种模式

### 检索（实现前）

1. 读 `docs/_index.md` → 找到相关文档
2. 读 `docs/architecture.md` → 理解接口体系、Provider 规范、数据流
3. 读 `docs/providers.md` → 参考同级 provider 实现模式
4. **Provider 相关** → 读 `docs/api/{provider}/`（SDK 用法）+ `docs/reference/{provider}/`（官方 API 真值）
5. 对照 `sdk-architecture` 边界矩阵确认合规
6. 汇报约束后给出实现建议

### 沉淀（实现后）— 体系维护，不是追加

1. **去重** — 搜 `docs/` 全量已有内容，确认新增不重复；重复则合并而非新增文件
2. **合并** — 同一主题多个文档有重叠时合并，删除冗余文件，更新 `_index.md`
3. **更新** — 接口/Provider/配置变更 → 更新对应文档，不追加段落，重写以保持信息密度
4. **废弃** — 标记或删除过时文档，更新所有引用
5. **索引** — 更新 `docs/_index.md`，确认双向引用完整

### 检查（手动触发）

```
□ 搜重复内容：同一概念是否在多个文档中出现
□ 信息密度：是否有大段叙述可压缩为列表/矩阵
□ 死链/死引用：_index.md 引用的文档是否存在
□ 索引完整：docs/_index.md 是否覆盖所有 docs/*.md
□ .claude/_index.md 是否覆盖全部 skills/agents/output-styles
```

```bash
# 列出所有 _index.md
find . -name "_index.md" -not -path "./.git/*" -not -path "./.codegraph/*" -not -path "./graphify-out/*"

# 检查有 3+ .go 文件却无 _index.md 的目录
find . -type d -not -path "./.git/*" -not -path "./.codegraph/*" \
  -not -path "./docs/*" -not -path "./graphify-out/*" \
  -not -path "./.specify/*" -not -path "./.claude/*" \
  -exec sh -c 'n=$(find "$1" -maxdepth 1 -name "*.go" | wc -l)
  [ "$n" -ge 3 ] && [ ! -f "$1/_index.md" ] && echo "缺失: $1 ($n files)"' _ {} \;
```

### 维护（持续）

| 检查项 | 规则 |
| ------ | ---- |
| 去重   | 同一概念不出现在 2+ 文档中，重复内容合并 |
| 密度   | 每个文档 ≤ 80 行；优先列表、矩阵、一句话高密度 |
| 引用   | `_index.md` 引用的文档必须存在；新增文档必须被引用 |
| 废弃   | 代码已删而文档还在的，标记 `> ⚠️ 待更新` 或删除 |

## 注意事项

- 只记引用，不重复代码文档
- 设计文档存在才引用，目录变更时修正相对路径
- 文件少于 3 个的目录可暂不创建
- **沉淀 = 体系维护**：去重 > 合并 > 更新 > 废弃，不盲目追加
