# llm-sdk-go 项目命令入口 — just --list 查看全部

# ---- 开发 ----

# 编译 llm-tools 代码生成二进制
gen-tools:
    go build -o _tools/llm-tools ./cmd/llm-tools/

# 验证编译
build: gen-tools
    go build ./...

# tidy 依赖
tidy:
    go mod tidy

# ---- 测试 & 检查 ----

# 全量验收：lint + test + build
all: gen-tools test build

# 静态检查（自动修复）
lint:
    golangci-lint run --fix ./...

# lint + test
test: lint
    go test -race ./...

# 仅测试（跳过 lint，更快）
test-only:
    go test -v -race ./...

# 仅单元测试（跳过集成测试）
test-unit:
    go test -v -race -short ./...

# 代码生成相关测试
test-gen:
    go test -count=1 ./internal/codegen/...
    go test -count=1 ./cmd/llm-tools/...

# 清理测试缓存
clean:
    go clean -testcache

# ---- 文档 ----

# 自动生成缺失的 _index.md
gen-index:
    go run ./cmd/tools index gen

# 检查 _index.md 覆盖率
check-docs:
    go run ./cmd/tools index check

# ---- Git Hooks ----

# 安装 git pre-commit hook
setup-hooks:
    printf '#!/bin/sh\nexec just check-docs\n' > .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    @echo "pre-commit hook installed -> just check-docs"
