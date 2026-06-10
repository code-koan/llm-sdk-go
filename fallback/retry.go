package fallback

import (
	"errors"
	"math/rand"
	"time"

	sdkerrors "github.com/code-koan/llm-sdk-go/errors"
)

// Default configuration for DefaultRetryPolicy.
const (
	defaultMaxAttempts = 5
	defaultBaseBackoff = 1 * time.Second
	defaultMaxBackoff  = 30 * time.Second
)

// RetryPolicy determines how to respond to a failed provider call.
type RetryPolicy interface {
	// ShouldRetry checks whether the current provider should be retried after
	// a failure. attempt is 0-based (0 means the first attempt just failed).
	//
	// When retry is false the Router moves to the next provider. When retry
	// is true the Router waits for wait and calls the same provider again.
	ShouldRetry(attempt int, err error) (wait time.Duration, retry bool)
}

// DefaultRetryPolicy classifies errors and applies exponential backoff for
// transient failures.
//
// Classification:
//   - Rate limit / model not found → switch provider immediately
//   - Authentication → switch provider (different keys may work)
//   - Request-level errors (invalid request, context length, content filter,
//     missing API key, unsupported param) → switch provider
//   - All other errors (network, server, provider) → exponential backoff
//     on the same provider
type DefaultRetryPolicy struct {
	// MaxAttempts is the maximum number of attempts per provider, including
	// the initial call. The default is 5.
	MaxAttempts int

	// BaseBackoff is the initial backoff duration. It doubles on each retry.
	// The default is 1 second.
	BaseBackoff time.Duration

	// MaxBackoff is the maximum backoff duration. The default is 30 seconds.
	MaxBackoff time.Duration

	// Jitter adds randomness to backoff to prevent thundering herd.
	// A jitter of 0.5 means the computed backoff is randomized to
	// [backoff, backoff*1.5). The default is 0 (no jitter).
	Jitter float64
}

// NewDefaultRetryPolicy creates a DefaultRetryPolicy with sensible defaults.
func NewDefaultRetryPolicy() *DefaultRetryPolicy {
	return &DefaultRetryPolicy{
		MaxAttempts: defaultMaxAttempts,
		BaseBackoff: defaultBaseBackoff,
		MaxBackoff:  defaultMaxBackoff,
	}
}

// ShouldRetry classifies the error and returns whether to retry the same
// provider, and how long to wait before doing so.
func (p *DefaultRetryPolicy) ShouldRetry(attempt int, err error) (time.Duration, bool) {
	if attempt >= p.MaxAttempts {
		return 0, false
	}

	// Rate limit and model-not-found: different provider may help.
	if errors.Is(err, sdkerrors.ErrRateLimit) || errors.Is(err, sdkerrors.ErrModelNotFound) {
		return 0, false
	}

	// Auth: different provider may use a valid key.
	if errors.Is(err, sdkerrors.ErrAuthentication) {
		return 0, false
	}

	// Request-level errors: the same request will likely fail on any provider,
	// but still try the next one since different providers may handle
	// parameters differently.
	if errors.Is(err, sdkerrors.ErrInvalidRequest) ||
		errors.Is(err, sdkerrors.ErrContextLength) ||
		errors.Is(err, sdkerrors.ErrContentFilter) ||
		errors.Is(err, sdkerrors.ErrMissingAPIKey) ||
		errors.Is(err, sdkerrors.ErrUnsupportedParam) {
		return 0, false
	}

	// All other errors (network timeouts, 5xx server errors, ErrProvider,
	// or unrecognized errors): retry same provider with exponential backoff.
	backoff := p.BaseBackoff * time.Duration(1<<uint(attempt))
	if backoff > p.MaxBackoff {
		backoff = p.MaxBackoff
	}
	if p.Jitter > 0 {
		jitter := time.Duration(float64(backoff) * p.Jitter * rand.Float64())
		backoff += jitter
	}
	return backoff, true
}
