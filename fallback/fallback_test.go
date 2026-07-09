package fallback

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/config"
	sdkerrors "github.com/code-koan/llm-sdk-go/errors"
	"github.com/code-koan/llm-sdk-go/internal/testutil"
	"github.com/code-koan/llm-sdk-go/providers"
)

// immediateRetry is a RetryPolicy that retries immediately (no wait) for all
// errors, to avoid sleeping in tests.
type immediateRetry struct{}

func (immediateRetry) ShouldRetry(_ int, _ error) (time.Duration, bool) {
	return 0, true
}

// longWaitRetry is a RetryPolicy that retries with a long wait, used in
// context cancellation tests so that ctx.Done() is the only ready channel
// when the select runs.
type longWaitRetry struct{}

func (longWaitRetry) ShouldRetry(_ int, _ error) (time.Duration, bool) {
	return time.Second, true
}

func TestNew_EmptyProviders(t *testing.T) {
	t.Parallel()

	_, err := New(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one provider")

	_, err = New([]providers.Provider{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one provider")
}

func TestNew_Defaults(t *testing.T) {
	t.Parallel()

	p := testutil.NewMockProvider()
	r, err := New([]providers.Provider{p})
	require.NoError(t, err)
	require.NotNil(t, r.selector)
	require.NotNil(t, r.retryPolicy)
	require.Equal(t, defaultMaxAttemptsPerProvider, r.maxAttemptsPerProvider)
	require.Equal(t, 1, len(r.providers))
}

func TestNew_InvalidOptions(t *testing.T) {
	t.Parallel()

	p := testutil.NewMockProvider()

	_, err := New([]providers.Provider{p}, WithSelector(nil))
	require.Error(t, err)

	_, err = New([]providers.Provider{p}, WithRetryPolicy(nil))
	require.Error(t, err)

	_, err = New([]providers.Provider{p}, WithMaxAttemptsPerProvider(0))
	require.Error(t, err)

	_, err = New([]providers.Provider{p}, WithMaxAttemptsPerProvider(-1))
	require.Error(t, err)

	_, err = New([]providers.Provider{p}, WithLogger(nil))
	require.Error(t, err)
}

func TestRouter_Name(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.NameFunc = func() string { return "alpha" }
	b := testutil.NewMockProvider()
	b.NameFunc = func() string { return "beta" }

	r, err := New([]providers.Provider{a, b})
	require.NoError(t, err)
	require.Equal(t, "fallback[alpha,beta]", r.Name())
}

func TestRouter_Capabilities(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.CapabilitiesFunc = func() providers.Capabilities {
		return providers.Capabilities{
			Completion:          true,
			CompletionStreaming: true,
			CompletionTools:     true,
			Embedding:           true,
			ListModels:          true,
		}
	}
	b := testutil.NewMockProvider()
	b.CapabilitiesFunc = func() providers.Capabilities {
		return providers.Capabilities{
			Completion:          true,
			CompletionStreaming: true,
			CompletionTools:     false, // no tools
			Embedding:           false, // no embedding
			ListModels:          true,
		}
	}

	r, err := New([]providers.Provider{a, b})
	require.NoError(t, err)

	caps := r.Capabilities()
	require.True(t, caps.Completion)
	require.True(t, caps.CompletionStreaming)
	require.False(t, caps.CompletionTools, "AND of true && false = false")
	require.False(t, caps.Embedding, "AND of true && false = false")
	require.True(t, caps.ListModels)

	// New capability fields — AND logic: seed=true, a=true, b=false → false
	require.False(t, caps.AsyncGeneration, "AND of true && false = false")
	require.False(t, caps.CompletionAudio, "AND of true && false = false")
	require.False(t, caps.CompletionVideo, "AND of true && false = false")
	require.False(t, caps.STT, "AND of true && false = false")
	require.False(t, caps.TTS, "AND of true && false = false")
}

func TestRouter_Completion_FirstSucceeds(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	r, err := New([]providers.Provider{a})
	require.NoError(t, err)

	resp, err := r.Completion(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, "Hello World", resp.Choices[0].Message.Content)
	require.Len(t, a.CompletionCalls, 1)
}

func TestRouter_Completion_FallbackOnRateLimit(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.NameFunc = func() string { return "a" }
	a.CompletionFunc = func(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
		return nil, sdkerrors.NewRateLimitError("a", fmt.Errorf("rate limited"))
	}

	b := testutil.NewMockProvider()
	b.NameFunc = func() string { return "b" }
	b.CompletionFunc = func(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
		return testutil.MockChatCompletion("hello from b"), nil
	}

	r, err := New(
		[]providers.Provider{a, b},
		WithSelector(NewRoundRobinSelector()), // deterministic order
		WithMaxAttemptsPerProvider(1),
	)
	require.NoError(t, err)

	resp, err := r.Completion(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, "hello from b", resp.Choices[0].Message.Content)
	require.Len(t, a.CompletionCalls, 1)
	require.Len(t, b.CompletionCalls, 1)
}

func TestRouter_Completion_AllFail(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.NameFunc = func() string { return "a" }
	a.CompletionFunc = func(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
		return nil, sdkerrors.NewRateLimitError("a", fmt.Errorf("rate limited"))
	}

	b := testutil.NewMockProvider()
	b.NameFunc = func() string { return "b" }
	b.CompletionFunc = func(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
		return nil, sdkerrors.NewAuthenticationError("b", fmt.Errorf("unauthorized"))
	}

	r, err := New(
		[]providers.Provider{a, b},
		WithSelector(NewRoundRobinSelector()),
		WithMaxAttemptsPerProvider(1),
	)
	require.NoError(t, err)

	resp, err := r.Completion(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})
	require.Nil(t, resp)
	require.Error(t, err)

	var allFailed *AllFailedError
	require.ErrorAs(t, err, &allFailed)
	require.True(t, errors.Is(err, sdkerrors.ErrAuthentication)) // LastError unwraps to auth
}

func TestRouter_Completion_RetrySameProvider(t *testing.T) {
	t.Parallel()

	callCount := 0
	a := testutil.NewMockProvider()
	a.NameFunc = func() string { return "a" }
	a.CompletionFunc = func(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
		callCount++
		if callCount == 1 {
			return nil, sdkerrors.ErrProvider
		}
		return testutil.MockChatCompletion("success on retry"), nil
	}

	r, err := New(
		[]providers.Provider{a},
		WithRetryPolicy(immediateRetry{}),
		WithMaxAttemptsPerProvider(3),
	)
	require.NoError(t, err)

	resp, err := r.Completion(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)
	require.Equal(t, "success on retry", resp.Choices[0].Message.Content)
	require.Len(t, a.CompletionCalls, 2)
}

func TestRouter_Completion_ContextCancelled(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.CompletionFunc = func(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
		return nil, sdkerrors.ErrProvider
	}

	// longWaitRetry returns a 1s wait — ctx.Done() becomes the only ready
	// channel once cancelled, making the test deterministic.
	r, err := New(
		[]providers.Provider{a},
		WithRetryPolicy(longWaitRetry{}),
		WithMaxAttemptsPerProvider(5),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay (well before the 1s wait elapses).
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err = r.Completion(ctx, providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestRouter_Completion_ContextAlreadyCancelled(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	r, err := New([]providers.Provider{a})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = r.Completion(ctx, providers.CompletionParams{
		Model: "test",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "hi"},
		},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestRouter_CompletionStream_FirstSucceeds(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	r, err := New([]providers.Provider{a})
	require.NoError(t, err)

	chunks, errs := r.CompletionStream(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})

	// Collect all chunks.
	var allChunks []providers.ChatCompletionChunk
	for chunk := range chunks {
		allChunks = append(allChunks, chunk)
	}

	// Should get 3 chunks (role, content, finish).
	require.Len(t, allChunks, 3)

	// Verify no errors.
	select {
	case err := <-errs:
		require.NoError(t, err)
	default:
	}
}

func TestRouter_CompletionStream_FallbackBeforeFirstChunk(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.NameFunc = func() string { return "a" }
	a.CompletionStreamFunc = func(ctx context.Context, params providers.CompletionParams) (<-chan providers.ChatCompletionChunk, <-chan error) {
		chunks := make(chan providers.ChatCompletionChunk)
		errs := make(chan error, 1)
		go func() {
			defer close(chunks)
			defer close(errs)
			errs <- sdkerrors.NewRateLimitError("a", fmt.Errorf("rate limited"))
		}()
		return chunks, errs
	}

	b := testutil.NewMockProvider()
	b.NameFunc = func() string { return "b" }

	r, err := New(
		[]providers.Provider{a, b},
		WithSelector(NewRoundRobinSelector()),
		WithMaxAttemptsPerProvider(1),
	)
	require.NoError(t, err)

	chunks, errs := r.CompletionStream(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})

	var allChunks []providers.ChatCompletionChunk
	for chunk := range chunks {
		allChunks = append(allChunks, chunk)
	}
	require.Len(t, allChunks, 3) // from provider b

	select {
	case err := <-errs:
		require.NoError(t, err)
	default:
	}
}

func TestRouter_CompletionStream_AllFail(t *testing.T) {
	t.Parallel()

	makeFailingProvider := func(name string) *testutil.MockProvider {
		p := testutil.NewMockProvider()
		p.NameFunc = func() string { return name }
		p.CompletionStreamFunc = func(ctx context.Context, params providers.CompletionParams) (<-chan providers.ChatCompletionChunk, <-chan error) {
			chunks := make(chan providers.ChatCompletionChunk)
			errs := make(chan error, 1)
			go func() {
				defer close(chunks)
				defer close(errs)
				errs <- sdkerrors.NewAuthenticationError(name, fmt.Errorf("unauthorized"))
			}()
			return chunks, errs
		}
		return p
	}

	a := makeFailingProvider("a")
	b := makeFailingProvider("b")

	r, err := New(
		[]providers.Provider{a, b},
		WithSelector(NewRoundRobinSelector()),
		WithMaxAttemptsPerProvider(1),
	)
	require.NoError(t, err)

	chunks, errs := r.CompletionStream(context.Background(), providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})

	var allChunks []providers.ChatCompletionChunk
	for chunk := range chunks {
		allChunks = append(allChunks, chunk)
	}
	require.Empty(t, allChunks)

	var streamErr error
	select {
	case e := <-errs:
		streamErr = e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
	require.Error(t, streamErr)

	var allFailed *AllFailedError
	require.ErrorAs(t, streamErr, &allFailed)
}

func TestRouter_CompletionStream_ContextCancel(t *testing.T) {
	t.Parallel()

	a := testutil.NewMockProvider()
	a.CompletionStreamFunc = func(ctx context.Context, params providers.CompletionParams) (<-chan providers.ChatCompletionChunk, <-chan error) {
		chunks := make(chan providers.ChatCompletionChunk)
		errs := make(chan error, 1)
		go func() {
			defer close(chunks)
			defer close(errs)
			errs <- sdkerrors.ErrProvider
		}()
		return chunks, errs
	}

	// longWaitRetry returns a 1s wait — ctx.Done() becomes the only ready
	// channel once cancelled, making the test deterministic.
	r, err := New(
		[]providers.Provider{a},
		WithRetryPolicy(longWaitRetry{}),
		WithMaxAttemptsPerProvider(5),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	chunks, errs := r.CompletionStream(ctx, providers.CompletionParams{
		Model:    "test",
		Messages: []providers.Message{{Role: providers.RoleUser, Content: "hi"}},
	})

	// Drain chunks.
	for range chunks {
	}

	var streamErr error
	select {
	case e := <-errs:
		streamErr = e
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
	require.ErrorIs(t, streamErr, context.Canceled)
}

func TestRouter_WithLogger(t *testing.T) {
	// Not parallel: testLogger tracks state.

	var logged bool
	tl := &testLogger{
		debugf: func(msg string, fields ...config.Field) {
			logged = true
			require.Equal(t, "completion attempt", msg)
		},
	}

	p := testutil.NewMockProvider()
	r, err := New([]providers.Provider{p}, WithLogger(tl))
	require.NoError(t, err)

	_, err = r.Completion(context.Background(), providers.CompletionParams{
		Model: "test",
		Messages: []providers.Message{
			{Role: providers.RoleUser, Content: "hi"},
		},
	})
	require.NoError(t, err)
	require.True(t, logged, "logger should have been called")
}

// testLogger is a simple config.Logger for testing.
type testLogger struct {
	debugf func(string, ...config.Field)
}

func (l *testLogger) Debug(msg string, fields ...config.Field) {
	if l.debugf != nil {
		l.debugf(msg, fields...)
	}
}

func (l *testLogger) Info(string, ...config.Field)  {}
func (l *testLogger) Warn(string, ...config.Field)  {}
func (l *testLogger) Error(string, ...config.Field) {}

func TestAllFailedError_Unwrap(t *testing.T) {
	t.Parallel()

	orig := fmt.Errorf("something broke")
	e := &AllFailedError{
		Errors:    []error{orig},
		LastError: orig,
	}

	require.Equal(t, "fallback: all providers failed (last: something broke)", e.Error())
	require.True(t, errors.Is(e, orig))
}
