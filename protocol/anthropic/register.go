package anthropic

import (
	"github.com/code-koan/llm-sdk-go/protocol"
	"github.com/code-koan/llm-sdk-go/providers"
)

func init() {
	protocol.Register(&protocol.ProtocolDef{
		Name: protocol.Anthropic,
		Detect: func(req any) bool {
			_, ok := req.(*MessageRequest)
			return ok
		},
		ToSDK: func(req any) (*providers.CompletionParams, error) {
			return ToCompletionParams(req.(*MessageRequest))
		},
		FromSDK: func(completion *providers.ChatCompletion, req any) any {
			return FromCompletion(completion, req.(*MessageRequest))
		},
		NewStream: func() protocol.StreamAdapter {
			return &hubStreamAdapter{inner: NewStreamAdapter()}
		},
	})
}
