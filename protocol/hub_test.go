package protocol_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/internal/testutil"
	"github.com/code-koan/llm-sdk-go/protocol"
	protocolanthropic "github.com/code-koan/llm-sdk-go/protocol/anthropic"
)

func TestInput_Valid(t *testing.T) {
	t.Parallel()
	mock := testutil.NewMockProvider()
	req := &protocolanthropic.MessageRequest{
		Model:     "claude-3",
		MaxTokens: 100,
		Messages:  []protocolanthropic.Message{{Role: "user", Content: "hello"}},
	}
	resp, err := protocol.Using(mock).Input(req).Want(protocol.Anthropic).Call(context.Background())
	require.NoError(t, err)
	require.Equal(t, protocol.Protocol("anthropic"), resp.Protocol)
	require.NotNil(t, resp.Data)
}

func TestInput_UnknownType(t *testing.T) {
	t.Parallel()
	mock := testutil.NewMockProvider()
	req := struct{ X string }{X: "y"}
	_, err := protocol.Using(mock).Input(req).Want(protocol.Anthropic).Call(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown protocol")
}
