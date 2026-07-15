# llm-sdk-go Constitution

## Core Principles

### I. Unified Provider Interface (NON-NEGOTIABLE)
`providers.Provider` is the single entry point for all LLM access. Every provider MUST implement `Completion()` and `CompletionStream()`. Optional capabilities via type-asserted interfaces only — never add methods to the base Provider interface without full-scope impact assessment.

### II. OpenAI-Compatible Output (NON-NEGOTIABLE)
All responses normalize to OpenAI format. ChatCompletion, ChatCompletionChunk, FinishReason, Role, Usage — these are the canonical types. Provider-native types stay in `protocol/` as intermediate bridges, never leak to consumers.

### III. Zero Unnecessary Abstraction
No interface with one implementation. No factory for one product. No config field that never changes. Every abstraction must carry its weight — if it doesn't reduce duplication or enable polymorphism across ≥2 concrete types, it doesn't exist.

### IV. Provider Isolation
Providers under `providers/*/` must not import each other. Each provider is an independent implementation of the shared interface. Shared logic goes in `providers/` (types, message, model) or `internal/` — never in another provider's package.

### V. Streaming Safety (NON-NEGOTIABLE)
Every goroutine-to-channel send MUST use `select { case ch <- v: case <-ctx.Done(): }`. Streaming goroutines manage their own lifecycle via context. Consumer abandonment must never leak a goroutine.

### VI. Error Normalization
All provider errors normalize through `errors.As` with SDK typed errors (`ErrRateLimit`, `ErrAuthentication`, etc.). String matching is forbidden unless the provider's error has no structured type — and even then, document why.

### VII. Test Isolation
No real network requests in tests. Use `httptest.Server`, fake clients, or skip gracefully. Integration tests reference `docs/reference/{provider}/` as ground truth for expected request/response shapes.

## Development Workflow

- **sdk-architecture** is the architectural authority — all changes must comply with its boundaries and invariants
- **deep-coding** governs the implementation workflow: speckit chain (specify → clarify → plan → tasks → implement → converge) → review gate → reflect
- **provider-adpter** governs Provider-specific work: Phase 0 (doc archive) → Phase 1 (design gate) → Phase 2 (TDD) → Phase 3 (checklist) → Phase 4 (test coverage) → Phase 5 (doc close)

## Governance

- Constitution supersedes all other practices. Conflicting rules → constitution wins.
- Amendments require: documented rationale, impact assessment on all existing providers, migration plan if breaking.
- All PR reviews must verify constitutional compliance. Complexity must be justified against Principle III.
- `sdk-architecture` SKILL.md is the living specification derived from this constitution. Keep them in sync.

**Version**: 1.0.0 | **Ratified**: 2026-07-15 | **Last Amended**: 2026-07-15
