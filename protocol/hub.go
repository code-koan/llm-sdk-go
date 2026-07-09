// Package protocol provides a central routing hub that chains .Input().Want() to
// transparently convert between API protocols (Anthropic, OpenAI, Responses, etc.)
// through a single provider interface.
package protocol

import (
	"context"
	"fmt"
	"sync"

	"github.com/code-koan/llm-sdk-go/providers"
)

// Protocol identifies a specific API protocol (e.g., "anthropic", "openai").
type Protocol string

const (
	Anthropic Protocol = "anthropic"
	OpenAI    Protocol = "openai"
	Responses Protocol = "responses"
)

// ProtocolDef defines how to convert between a protocol's native types and the
// SDK's internal representation.
type ProtocolDef struct {
	Name      Protocol
	Detect    func(req any) bool
	ToSDK     func(req any) (*providers.CompletionParams, error)
	FromSDK   func(completion *providers.ChatCompletion, req any) any
	NewStream func() StreamAdapter
}

// StreamAdapter converts SDK streaming chunks into protocol-specific stream events.
type StreamAdapter interface {
	Adapt(chunk providers.ChatCompletionChunk) []any
	Flush() []any
}

var (
	mu       sync.RWMutex
	registry = map[Protocol]*ProtocolDef{}
)

// Register registers a protocol definition with the central hub.
func Register(def *ProtocolDef) {
	mu.Lock()
	defer mu.Unlock()
	registry[def.Name] = def
}

// NativeProvider is an optional interface for providers that can handle
// protocol-native requests directly, bypassing the SDK conversion layer.
type NativeProvider interface {
	CallNative(ctx context.Context, req any) (any, error)
}

// NativeStreamProvider is an optional interface for providers that can handle
// protocol-native streaming requests directly.
type NativeStreamProvider interface {
	CallNativeStream(ctx context.Context, req any) (<-chan any, <-chan error)
}

// Response wraps the result of a protocol-routed call.
type Response struct {
	Protocol Protocol
	Data     any
}

// Chain is the entry point for the protocol routing API.
type Chain struct {
	provider providers.Provider
}

// Using creates a new Chain bound to the given provider.
func Using(p providers.Provider) *Chain {
	return &Chain{provider: p}
}

// InputStep is the intermediate step after specifying the input request.
type InputStep struct {
	provider  providers.Provider
	input     any
	fromProto *ProtocolDef
	err       error
}

// Input starts a protocol chain with the given request and detects its protocol.
func (c *Chain) Input(req any) *InputStep {
	def, err := detectProtocol(req)
	return &InputStep{
		provider:  c.provider,
		input:     req,
		fromProto: def,
		err:       err,
	}
}

// CallStep is the intermediate step after specifying the target protocol.
type CallStep struct {
	provider  providers.Provider
	input     any
	fromProto *ProtocolDef
	toProto   *ProtocolDef
	inputErr  error
}

// Want sets the target protocol for the conversion chain.
func (s *InputStep) Want(target Protocol) *CallStep {
	mu.RLock()
	def := registry[target]
	mu.RUnlock()
	return &CallStep{
		provider:  s.provider,
		input:     s.input,
		fromProto: s.fromProto,
		toProto:   def,
		inputErr:  s.err,
	}
}

// Call performs a synchronous completion, routing through the appropriate
// conversion path.
func (s *CallStep) Call(ctx context.Context) (*Response, error) {
	if s.inputErr != nil {
		return nil, s.inputErr
	}
	if s.toProto == nil {
		return nil, fmt.Errorf("unknown target protocol")
	}

	// Same-protocol path: check if provider supports native interface.
	if s.fromProto.Name == s.toProto.Name {
		if np, ok := s.provider.(NativeProvider); ok {
			data, err := np.CallNative(ctx, s.input)
			if err != nil {
				return nil, err
			}
			return &Response{Protocol: s.toProto.Name, Data: data}, nil
		}
	}

	// Cross-protocol path: ToSDK -> Completion -> FromSDK.
	params, err := s.fromProto.ToSDK(s.input)
	if err != nil {
		return nil, fmt.Errorf("convert input: %w", err)
	}
	completion, err := s.provider.Completion(ctx, *params)
	if err != nil {
		return nil, err
	}
	data := s.toProto.FromSDK(completion, s.input)
	return &Response{Protocol: s.toProto.Name, Data: data}, nil
}

// CallStream performs a streaming completion, routing through the appropriate
// conversion path.
func (s *CallStep) CallStream(ctx context.Context) (<-chan any, <-chan error) {
	events := make(chan any)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		if s.inputErr != nil {
			errs <- s.inputErr
			return
		}
		if s.toProto == nil {
			errs <- fmt.Errorf("unknown target protocol")
			return
		}

		// Same-protocol native streaming.
		if s.fromProto.Name == s.toProto.Name {
			if nsp, ok := s.provider.(NativeStreamProvider); ok {
				nativeEvents, nativeErrs := nsp.CallNativeStream(ctx, s.input)
				forwardStream(ctx, events, errs, nativeEvents, nativeErrs)
				return
			}
		}

		// Cross-protocol streaming.
		params, err := s.fromProto.ToSDK(s.input)
		if err != nil {
			errs <- fmt.Errorf("convert input: %w", err)
			return
		}
		params.Stream = true
		chunks, chunkErrs := s.provider.CompletionStream(ctx, *params)

		adapter := s.toProto.NewStream()
		for {
			select {
			case chunk, ok := <-chunks:
				if !ok {
					for _, e := range adapter.Flush() {
						select {
						case events <- e:
						case <-ctx.Done():
							return
						}
					}
					return
				}
				for _, e := range adapter.Adapt(chunk) {
					select {
					case events <- e:
					case <-ctx.Done():
						return
					}
				}
			case e, ok := <-chunkErrs:
				if !ok {
					chunkErrs = nil
					continue
				}
				if e != nil {
					errs <- e
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return events, errs
}

// forwardStream forwards events and errors from a native stream to the output
// channels, respecting context cancellation.
func forwardStream(
	ctx context.Context,
	events chan<- any,
	errs chan<- error,
	nativeEvents <-chan any,
	nativeErrs <-chan error,
) {
	for {
		select {
		case e, ok := <-nativeEvents:
			if !ok {
				return
			}
			select {
			case events <- e:
			case <-ctx.Done():
				return
			}
		case e, ok := <-nativeErrs:
			if !ok {
				nativeErrs = nil
				continue
			}
			if e != nil {
				select {
				case errs <- e:
				case <-ctx.Done():
				}
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// detectProtocol finds the registered ProtocolDef that matches the given request.
func detectProtocol(req any) (*ProtocolDef, error) {
	mu.RLock()
	defer mu.RUnlock()
	for _, def := range registry {
		if def.Detect(req) {
			return def, nil
		}
	}
	return nil, fmt.Errorf("unknown protocol: no registered protocol matches request of type %T", req)
}
