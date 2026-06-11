# Cache and Rate Limit

This page documents user-facing cache, attribution, per-request extension, and rate-limit behavior.

## Prompt caching

### OpenAI-compatible providers

OpenAI-compatible providers use provider-side automatic prompt caching when the upstream service supports it. There is no SDK request-side `cache_control` field for OpenAI-compatible caching.

Cache hits are surfaced in normalized usage:

```go
resp, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4.1",
    Messages: []llmsdk.Message{
        {Role: llmsdk.RoleUser, Content: "large repeated prompt..."},
    },
})
if err != nil {
    return err
}

fmt.Println("cached input tokens:", resp.Usage.CacheReadInputTokens)
```

Mapping:

- OpenAI `usage.prompt_tokens_details.cached_tokens` → `Usage.CacheReadInputTokens`
- `Usage.PromptTokens` remains total prompt input tokens, including cached input tokens.

### Anthropic explicit cache control

Anthropic prompt caching is explicit. Set `CacheControl` on the request element that should become a cache breakpoint:

- `CompletionParams.CacheControl` for top-level request cache control
- `ContentPart.CacheControl` for a text/content block
- `Tool.CacheControl` for tool definitions

Anthropic's default ephemeral cache TTL is 5 minutes. Set `TTL: llmsdk.CacheControlTTL1h` when you need the optional 1 hour TTL.

```go
resp, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "claude-sonnet-4-20250514",
    Messages: []llmsdk.Message{{
        Role: llmsdk.RoleUser,
        Content: []llmsdk.ContentPart{{
            Type: "text",
            Text: "large stable context...",
            CacheControl: &llmsdk.CacheControlParam{
                Type: llmsdk.CacheControlTypeEphemeral,
                TTL:  llmsdk.CacheControlTTL1h,
            },
        }, {
            Type: "text",
            Text: "question about the context",
        }},
    }},
})
```

Anthropic usage mapping:

- `usage.cache_read_input_tokens` → `Usage.CacheReadInputTokens`
- `usage.cache_creation_input_tokens` → `Usage.CacheCreationInputTokens`
- `usage.cache_creation.ephemeral_1h_input_tokens` → `Usage.CacheCreation.Ephemeral1hInputTokens`
- `usage.cache_creation.ephemeral_5m_input_tokens` → `Usage.CacheCreation.Ephemeral5mInputTokens`
- `Usage.PromptTokens` is total prompt input tokens: Anthropic `input_tokens + cache_creation_input_tokens + cache_read_input_tokens`.

## User attribution

Use `llmsdk.WithUserID` to set a provider default user identifier. Set `CompletionParams.User` to override it for a single request.

```go
provider, err := openai.New(
    llmsdk.WithAPIKey("sk-..."),
    llmsdk.WithUserID("tenant-abc"),
)

resp, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4.1",
    Messages: messages,
    User: "tenant-override", // optional per-request override
})
```

Provider mapping:

- OpenAI-compatible providers: request body `user`
- Anthropic: request body `metadata.user_id`

## Per-request headers and body extension

`CompletionParams` supports request-local escape hatches for provider-specific features:

```go
resp, err := provider.Completion(ctx, llmsdk.CompletionParams{
    Model: "gpt-4.1",
    Messages: messages,
    Headers: map[string]string{
        "OpenAI-Project": "proj_xxx",
    },
    Extra: map[string]any{
        "prompt_cache_key": "tenant-abc-docs-v1",
    },
    OverrideBody: map[string]any{
        "temperature": 0.2,
    },
})
```

Rules:

- `Headers` adds provider request headers for that request only.
- `Extra` adds non-conflicting JSON body fields for that request.
- `OverrideBody` explicitly replaces generated JSON body fields for that request.
- The SDK does not mutate request bodies in a global HTTP transport and does not perform transport-level deep merge.
- Do not use `Extra` or `OverrideBody` for fields already modeled by `CompletionParams` unless you intentionally need provider-specific override behavior.

## Rate limit errors

Providers normalize rate-limit failures to `llmsdk.ErrRateLimit` and `*llmsdk.RateLimitError`.

```go
resp, err := provider.Completion(ctx, params)
if err != nil {
    if errors.Is(err, llmsdk.ErrRateLimit) {
        var rateErr *llmsdk.RateLimitError
        if errors.As(err, &rateErr) {
            fmt.Println("retry after seconds:", rateErr.RetryAfter)
            fmt.Println("rate limit headers:", rateErr.Headers)
        }
    }
    return err
}
```

`RateLimitError.RetryAfter` is populated from provider response headers when available. `RateLimitError.Headers` contains recognized provider rate-limit headers such as `Retry-After`, OpenAI `X-RateLimit-*`, and Anthropic `anthropic-ratelimit-*` headers.

The SDK does not automatically retry. Applications should decide retry policy, backoff, idempotency, and request cancellation behavior.

## Boundaries

- OpenAI-compatible prompt caching is automatic provider behavior; the SDK only maps returned cache usage.
- Anthropic cache control is supported only through the typed `CacheControl` fields above.
- Per-request `Headers`, `Extra`, and `OverrideBody` are request-local; they are not global provider configuration.
- No automatic retry, queueing, or token-bucket throttling is implemented by the SDK.
