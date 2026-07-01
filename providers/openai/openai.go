package openai

import (
	"github.com/code-koan/llm-sdk-go/config"
	"github.com/code-koan/llm-sdk-go/providers"
)

// Provider configuration constants.
const (
	defaultBaseURL = "https://api.openai.com/v1"
	envAPIKey      = "OPENAI_API_KEY"
	providerName   = "openai"
)

// Ensure Provider implements the required interfaces.
var (
	_ providers.CapabilityProvider = (*Provider)(nil)
	_ providers.EmbeddingProvider  = (*Provider)(nil)
	_ providers.ErrorConverter     = (*Provider)(nil)
	_ providers.ModelLister        = (*Provider)(nil)
	_ providers.Provider           = (*Provider)(nil)
)

// Provider implements the providers.Provider interface for OpenAI.
// It embeds CompatibleProvider which handles the OpenAI SDK integration.
type Provider struct {
	*CompatibleProvider
}

// New creates a new OpenAI provider.
func New(opts ...config.Option) (*Provider, error) {
	base, err := NewCompatible(CompatibleConfig{
		APIKeyEnvVar:   envAPIKey,
		Capabilities:   capabilities(),
		DefaultBaseURL: defaultBaseURL,
		Name:           providerName,
		RequireAPIKey:  true,
	}, opts...)
	if err != nil {
		return nil, err
	}

	return &Provider{CompatibleProvider: base}, nil
}

// capabilities returns the capabilities for the OpenAI provider.
func capabilities() providers.Capabilities {
	return providers.Capabilities{
		AsyncGeneration:     false,
		Completion:          true,
		CompletionAudio:     false,
		CompletionImage:     true,
		CompletionPDF:       false,
		CompletionReasoning: true,
		CompletionStreaming: true,
		CompletionTools:     true,
		CompletionVideo:     false,
		Embedding:           true,
		ListModels:          true,
		STT:                 false,
		TTS:                 false,
	}
}
