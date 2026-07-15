---
name: subagent-implementation-not-planning
description: Subagents default to producing plans instead of writing code — how to get them to implement
metadata:
  type: feedback
---

# Subagent Implementation vs Planning

## Problem

When subagents are launched with implementation tasks, they default to producing detailed plans instead of writing actual code. Even when the parent session has already produced a comprehensive plan, subagents treat "implement X" as "plan how to implement X."

Root causes:
1. Subagent prompts lack explicit "this is implementation, NOT planning" signals
2. Subagents in `isolation: "worktree"` mode may lack project context
3. No post-agent verification checking for actual file changes

## How to fix

### For task prompts, always include:
```
IMPLEMENTATION TASK (not planning). Directly edit/create files NOW.
Do not produce a plan — the plan is already approved.
```

### Post-agent verification:
- Check `git diff --stat` in the agent's worktree before accepting results
- If 0 files changed, the agent planned instead of implementing → resend with explicit "WRITE CODE NOW" instruction
- Prefer agents WITHOUT `isolation: "worktree"` for minor changes (<5 files)

**Why:** Agents consume significant tokens but produce zero code changes. This blocks the primary benefit of multi-agent orchestration.

**How to apply:** Before launching implementation agents, add the explicit signal as the first line of the prompt. After agent completion, verify file changes exist before proceeding.

See also: [[subagent-output-verification]], [[feedback-delegate-to-subagents]]
