package fallback

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/code-koan/llm-sdk-go/config"
	"github.com/code-koan/llm-sdk-go/providers"
)

const (
	defaultMaxAttemptsPerProvider = 2
)

// Interface assertions.
var (
	_ providers.Provider           = (*Router)(nil)
	_ providers.CapabilityProvider = (*Router)(nil)
)

// Router distributes requests across multiple LLM providers with configurable
// selection and retry policies. It implements providers.Provider so it can be
// used as a drop-in replacement anywhere a single provider is expected.
type Router struct {
	providers             []providers.Provider
	selector              Selector
	retryPolicy           RetryPolicy
	maxAttemptsPerProvider int
	logger                config.Logger
}

// Option configures a Router.
type Option func(*Router) error

// New creates a Router that distributes requests across the given providers.
// At least one provider is required. Defaults: RandomSelector,
// DefaultRetryPolicy, maxAttemptsPerProvider=2, no logging.
func New(backends []providers.Provider, opts ...Option) (*Router, error) {
	if len(backends) == 0 {
		return nil, fmt.Errorf("fallback: at least one provider is required")
	}

	r := &Router{
		providers:              slices.Clone(backends),
		selector:               NewRandomSelector(),
		retryPolicy:            NewDefaultRetryPolicy(),
		maxAttemptsPerProvider: defaultMaxAttemptsPerProvider,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(r); err != nil {
			return nil, fmt.Errorf("fallback: %w", err)
		}
	}

	return r, nil
}

// WithSelector sets the provider selection strategy. The default is
// RandomSelector.
func WithSelector(s Selector) Option {
	return func(r *Router) error {
		if s == nil {
			return fmt.Errorf("selector must not be nil")
		}
		r.selector = s
		return nil
	}
}

// WithRetryPolicy sets the retry strategy. The default is
// DefaultRetryPolicy.
func WithRetryPolicy(p RetryPolicy) Option {
	return func(r *Router) error {
		if p == nil {
			return fmt.Errorf("retry policy must not be nil")
		}
		r.retryPolicy = p
		return nil
	}
}

// WithMaxAttemptsPerProvider sets the maximum number of calls (initial +
// retries) to a single provider before moving to the next one. The default is
// 2. Must be at least 1.
func WithMaxAttemptsPerProvider(n int) Option {
	return func(r *Router) error {
		if n < 1 {
			return fmt.Errorf("max attempts per provider must be >= 1, got %d", n)
		}
		r.maxAttemptsPerProvider = n
		return nil
	}
}

// WithLogger sets the logger for debug-level attempt/failover messages. By
// default no log output is produced.
func WithLogger(l config.Logger) Option {
	return func(r *Router) error {
		if l == nil {
			return fmt.Errorf("logger must not be nil")
		}
		r.logger = l
		return nil
	}
}

// Name returns a composite name showing all providers in the pool.
func (r *Router) Name() string {
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = p.Name()
	}
	return "fallback[" + strings.Join(names, ",") + "]"
}

// Capabilities returns the logical AND of all provider capabilities.
func (r *Router) Capabilities() providers.Capabilities {
	if len(r.providers) == 0 {
		return providers.Capabilities{}
	}

	caps := providers.Capabilities{
		AsyncGeneration:     true,
		Completion:          true,
		CompletionAudio:     true,
		CompletionImage:     true,
		CompletionPDF:       true,
		CompletionReasoning: true,
		CompletionStreaming: true,
		CompletionTools:     true,
		CompletionVideo:     true,
		Embedding:           true,
		ListModels:          true,
		STT:                 true,
		TTS:                 true,
	}

	for _, p := range r.providers {
		cp, ok := p.(providers.CapabilityProvider)
		if !ok {
			continue
		}
		c := cp.Capabilities()
		caps.AsyncGeneration = caps.AsyncGeneration && c.AsyncGeneration
		caps.Completion = caps.Completion && c.Completion
		caps.CompletionAudio = caps.CompletionAudio && c.CompletionAudio
		caps.CompletionImage = caps.CompletionImage && c.CompletionImage
		caps.CompletionPDF = caps.CompletionPDF && c.CompletionPDF
		caps.CompletionReasoning = caps.CompletionReasoning && c.CompletionReasoning
		caps.CompletionStreaming = caps.CompletionStreaming && c.CompletionStreaming
		caps.CompletionTools = caps.CompletionTools && c.CompletionTools
		caps.CompletionVideo = caps.CompletionVideo && c.CompletionVideo
		caps.Embedding = caps.Embedding && c.Embedding
		caps.ListModels = caps.ListModels && c.ListModels
		caps.STT = caps.STT && c.STT
		caps.TTS = caps.TTS && c.TTS
	}

	return caps
}

// Completion calls providers in sequence according to the configured Selector
// and RetryPolicy. It returns the first successful response or an
// AllFailedError when every provider has been exhausted.
func (r *Router) Completion(ctx context.Context, params providers.CompletionParams) (*providers.ChatCompletion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	exclude := make(map[int]struct{})
	var lastErr error

	for {
		idx := r.selector.Select(r.providers, exclude)
		if idx < 0 {
			break
		}

		r.logDebug("completion attempt",
			config.Field{Key: "provider", Value: r.providers[idx].Name()},
			config.Field{Key: "model", Value: params.Model},
		)

		for attempt := 0; attempt < r.maxAttemptsPerProvider; attempt++ {
			resp, err := r.providers[idx].Completion(ctx, params)
			if err == nil {
				return resp, nil
			}
			lastErr = err

			wait, retry := r.retryPolicy.ShouldRetry(attempt, err)
			if !retry {
				r.logDebug("falling back to next provider",
					config.Field{Key: "provider", Value: r.providers[idx].Name()},
					config.Field{Key: "error", Value: err.Error()},
				)
				break
			}

			r.logDebug("retrying provider",
				config.Field{Key: "provider", Value: r.providers[idx].Name()},
				config.Field{Key: "attempt", Value: attempt + 1},
				config.Field{Key: "wait", Value: wait},
			)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		exclude[idx] = struct{}{}
	}

	return nil, &AllFailedError{LastError: lastErr}
}

// CompletionStream streams from providers with fallback on initial connection
// failure. Once the first chunk is delivered from a provider the stream is
// forwarded as-is; mid-stream failures are not retried.
func (r *Router) CompletionStream(ctx context.Context, params providers.CompletionParams) (<-chan providers.ChatCompletionChunk, <-chan error) {
	outChunks := make(chan providers.ChatCompletionChunk)
	outErrs := make(chan error, 1)

	go func() {
		defer close(outChunks)
		defer close(outErrs)

		exclude := make(map[int]struct{})
		var lastErr error

		for {
			idx := r.selector.Select(r.providers, exclude)
			if idx < 0 {
				outErrs <- &AllFailedError{LastError: lastErr}
				return
			}

			r.logDebug("stream attempt",
				config.Field{Key: "provider", Value: r.providers[idx].Name()},
				config.Field{Key: "model", Value: params.Model},
			)

		attemptLoop:
			for attempt := 0; attempt < r.maxAttemptsPerProvider; attempt++ {
				subChunks, subErrs := r.providers[idx].CompletionStream(ctx, params)

				select {
				case chunk, ok := <-subChunks:
					if ok {
						outChunks <- chunk
						r.forwardStream(ctx, outChunks, outErrs, subChunks, subErrs)
						return
					}
					// Chunk channel closed without delivering a chunk —
					// capture any error and move on.
					select {
					case err, ok := <-subErrs:
						if ok {
							lastErr = err
						}
					default:
					}
					break attemptLoop

				case err, ok := <-subErrs:
					if !ok {
						continue attemptLoop
					}
					lastErr = err

					wait, retry := r.retryPolicy.ShouldRetry(attempt, err)
					if !retry {
						r.logDebug("falling back to next provider",
							config.Field{Key: "provider", Value: r.providers[idx].Name()},
							config.Field{Key: "error", Value: err.Error()},
						)
						break attemptLoop
					}

					r.logDebug("retrying provider",
						config.Field{Key: "provider", Value: r.providers[idx].Name()},
						config.Field{Key: "attempt", Value: attempt + 1},
						config.Field{Key: "wait", Value: wait},
					)

					select {
					case <-ctx.Done():
						outErrs <- ctx.Err()
						return
					case <-time.After(wait):
					}

				case <-ctx.Done():
					outErrs <- ctx.Err()
					return
				}
			}

			exclude[idx] = struct{}{}
		}
	}()

	return outChunks, outErrs
}

// forwardStream copies all remaining chunks and errors from the provider
// channels to the output channels until both are closed or the context is
// cancelled. It does not close the output channels.
func (r *Router) forwardStream(
	ctx context.Context,
	outChunks chan<- providers.ChatCompletionChunk,
	outErrs chan<- error,
	inChunks <-chan providers.ChatCompletionChunk,
	inErrs <-chan error,
) {
	for inChunks != nil || inErrs != nil {
		select {
		case chunk, ok := <-inChunks:
			if !ok {
				inChunks = nil
				continue
			}
			outChunks <- chunk
		case err, ok := <-inErrs:
			if !ok {
				inErrs = nil
				continue
			}
			outErrs <- err
		case <-ctx.Done():
			return
		}
	}
}

// AllFailedError is returned when every provider in the Router has been tried
// and all failed. Use errors.Unwrap or errors.As to inspect the final error.
type AllFailedError struct {
	// Errors holds all errors encountered, in the order they occurred.
	Errors []error

	// LastError is the final error (also accessible via Unwrap).
	LastError error
}

// Error formats the error with the last error's message.
func (e *AllFailedError) Error() string {
	return fmt.Sprintf("fallback: all providers failed (last: %v)", e.LastError)
}

// Unwrap returns the last error for use with errors.Is / errors.As.
func (e *AllFailedError) Unwrap() error {
	return e.LastError
}

func (r *Router) logDebug(msg string, fields ...config.Field) {
	if r.logger != nil {
		r.logger.Debug(msg, fields...)
	}
}
