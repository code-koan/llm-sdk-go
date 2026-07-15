---
name: subagent-output-verification
description: Concrete protocol for verifying subagent output before accepting — covers constant names, signatures, file location, build, test
metadata:
  type: feedback
---

# Subagent Output Verification

Always run `go build ./...` and `go test -race ./...` on EVERY subagent's output before marking a task complete. Subagents routinely produce code with these specific error types:

1. **Wrong constant names** — e.g. using private constant when exported exists, or wrong package prefix
2. **Wrong method names** — e.g. old API name after rename. Verify against interface definition.
3. **Wrong file location** — agents may write to main repo path instead of worktree path
4. **Missing imports** — agents delete old constants but don't add new imports

**Why:** Subagents report "complete" but output has compilation errors. The build+test check catches 90% of these immediately.

**How to apply:** After every subagent completes a code-writing task:
1. `go build ./...` → fix any compilation errors
2. `go test -race ./...` → fix any test failures
3. Only THEN mark the task as complete

See also: [[subagent-implementation-not-planning]]
