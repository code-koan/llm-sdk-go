package responses

import (
	"github.com/code-koan/llm-sdk-go/protocol"
	"github.com/code-koan/llm-sdk-go/providers"
)

func init() {
	protocol.Register(&protocol.ProtocolDef{
		Name: protocol.Responses,
		Detect: func(req any) bool {
			_, ok := req.(*Request)
			return ok
		},
		ToSDK: func(req any) (*providers.CompletionParams, error) {
			return ToCompletionParams(req.(*Request))
		},
		FromSDK: func(completion *providers.ChatCompletion, req any) any {
			return FromCompletion(completion, req.(*Request))
		},
		NewStream: func() protocol.StreamAdapter {
			return &hubStreamAdapter{inner: NewStreamAdapter()}
		},
	})
}
