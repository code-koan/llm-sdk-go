package protocol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/providers"
)

// TestRequest is a dummy request type used for protocol detection tests.
type TestRequest struct {
	Message string
}

func init() {
	Register(&ProtocolDef{
		Name: "test",
		Detect: func(req any) bool {
			_, ok := req.(*TestRequest)
			return ok
		},
		ToSDK: func(req any) (*providers.CompletionParams, error) {
			return &providers.CompletionParams{
				Model:    "test-model",
				Messages: []providers.Message{{Role: "user", Content: req.(*TestRequest).Message}},
			}, nil
		},
		FromSDK: func(completion *providers.ChatCompletion, req any) any {
			return completion
		},
	})
}

func TestDetectProtocol_KnownType(t *testing.T) {
	t.Parallel()
	req := &TestRequest{Message: "hello"}
	def, err := detectProtocol(req)
	require.NoError(t, err)
	require.Equal(t, Protocol("test"), def.Name)
}

func TestDetectProtocol_UnknownType(t *testing.T) {
	t.Parallel()
	req := struct{ Foo string }{Foo: "bar"}
	_, err := detectProtocol(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown protocol")
}

func TestRegister(t *testing.T) {
	// Verify the registry is populated with the protocol registered via init().
	mu.RLock()
	def, ok := registry["test"]
	mu.RUnlock()
	require.True(t, ok, "expected test protocol to be registered")
	require.NotNil(t, def)
	require.NotNil(t, def.Detect)
	require.NotNil(t, def.ToSDK)
	require.NotNil(t, def.FromSDK)
}
