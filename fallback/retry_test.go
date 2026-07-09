package fallback

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdkerrors "github.com/code-koan/llm-sdk-go/errors"
)

func TestDefaultRetryPolicy_SwitchProviderErrors(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()

	tests := []struct {
		name string
		err  error
	}{
		{"rate limit", sdkerrors.ErrRateLimit},
		{"model not found", sdkerrors.ErrModelNotFound},
		{"authentication", sdkerrors.ErrAuthentication},
		{"invalid request", sdkerrors.ErrInvalidRequest},
		{"context length", sdkerrors.ErrContextLength},
		{"content filter", sdkerrors.ErrContentFilter},
		{"missing API key", sdkerrors.ErrMissingAPIKey},
		{"unsupported param", sdkerrors.ErrUnsupportedParam},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wait, retry := p.ShouldRetry(0, tc.err)
			require.False(t, retry, "should not retry same provider")
			require.Equal(t, time.Duration(0), wait, "should not wait when switching")
		})
	}
}

func TestDefaultRetryPolicy_TransientErrors(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()

	tests := []struct {
		name string
		err  error
	}{
		{"provider error", sdkerrors.ErrProvider},
		{"generic error", errors.New("network timeout")},
		{"wrapped error", fmt.Errorf("something failed: %w", errors.New("connection reset"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			wait, retry := p.ShouldRetry(0, tc.err)
			require.True(t, retry, "should retry same provider")
			require.Equal(t, defaultBaseBackoff, wait)
		})
	}
}

func TestDefaultRetryPolicy_ExponentialBackoff(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()

	// First retry: 1s
	wait, retry := p.ShouldRetry(0, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.Equal(t, 1*time.Second, wait)

	// Second retry: 2s
	wait, retry = p.ShouldRetry(1, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.Equal(t, 2*time.Second, wait)

	// Third retry: 4s
	wait, retry = p.ShouldRetry(2, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.Equal(t, 4*time.Second, wait)

	// Fourth retry: 8s
	wait, retry = p.ShouldRetry(3, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.Equal(t, 8*time.Second, wait)
}

func TestDefaultRetryPolicy_MaxBackoffCap(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()
	p.MaxAttempts = 8 // allow enough attempts to reach the cap

	// Attempt 4: 2^4 = 16s, not yet capped.
	wait, retry := p.ShouldRetry(4, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.Equal(t, 16*time.Second, wait)

	// Attempt 5: 2^5 = 32s, capped at 30s.
	wait, retry = p.ShouldRetry(5, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.Equal(t, defaultMaxBackoff, wait)
}

func TestDefaultRetryPolicy_MaxAttempts(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()
	p.MaxAttempts = 3

	// Attempts 0,1,2 should retry.
	_, retry := p.ShouldRetry(0, sdkerrors.ErrProvider)
	require.True(t, retry)
	_, retry = p.ShouldRetry(1, sdkerrors.ErrProvider)
	require.True(t, retry)
	_, retry = p.ShouldRetry(2, sdkerrors.ErrProvider)
	require.True(t, retry)

	// Attempt 3 → exhausted.
	_, retry = p.ShouldRetry(3, sdkerrors.ErrProvider)
	require.False(t, retry)
}

func TestDefaultRetryPolicy_Jitter(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()
	p.Jitter = 0.5

	// With jitter, backoff should be >= base and < base * (1 + jitter).
	wait, retry := p.ShouldRetry(0, sdkerrors.ErrProvider)
	require.True(t, retry)
	require.GreaterOrEqual(t, wait, defaultBaseBackoff)
	require.Less(t, wait, time.Duration(float64(defaultBaseBackoff)*1.5))

	// Jitter should not affect "switch provider" decisions.
	wait, retry = p.ShouldRetry(0, sdkerrors.ErrRateLimit)
	require.False(t, retry)
	require.Equal(t, time.Duration(0), wait)
}

func TestDefaultRetryPolicy_WrappedSentinelError(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()

	// A rate limit error wrapped in another error should still be detected
	// via errors.Is.
	wrappedErr := fmt.Errorf("call failed: %w", sdkerrors.NewRateLimitError("openai", fmt.Errorf("429")))
	wait, retry := p.ShouldRetry(0, wrappedErr)
	require.False(t, retry)
	require.Equal(t, time.Duration(0), wait)
}

func TestNewDefaultRetryPolicy_Defaults(t *testing.T) {
	t.Parallel()

	p := NewDefaultRetryPolicy()
	require.Equal(t, defaultMaxAttempts, p.MaxAttempts)
	require.Equal(t, defaultBaseBackoff, p.BaseBackoff)
	require.Equal(t, defaultMaxBackoff, p.MaxBackoff)
	require.Equal(t, float64(0), p.Jitter)
}
