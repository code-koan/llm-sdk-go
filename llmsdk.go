// Package llmsdk provides a unified interface for interacting with LLM providers.
//
// This package re-exports common types and configuration options from subpackages,
// allowing most use cases to work with just two imports:
//
//	import (
//	    llmsdk "github.com/code-koan/llm-sdk-go"
//	    "github.com/code-koan/llm-sdk-go/providers/openai"
//	)
//
//	provider, err := openai.New(llmsdk.WithAPIKey("sk-..."))
//	response, err := provider.Completion(ctx, llmsdk.CompletionParams{
//	    Model: "gpt-4o-mini",
//	    Messages: []llmsdk.Message{
//	        {Role: llmsdk.RoleUser, Content: "Hello!"},
//	    },
//	})
package llmsdk

import (
	"github.com/code-koan/llm-sdk-go/config"
	"github.com/code-koan/llm-sdk-go/errors"
	"github.com/code-koan/llm-sdk-go/fallback"
	"github.com/code-koan/llm-sdk-go/param"
	"github.com/code-koan/llm-sdk-go/protocol/anthropic"
	"github.com/code-koan/llm-sdk-go/protocol/responses"
	"github.com/code-koan/llm-sdk-go/providers"
	"github.com/code-koan/llm-sdk-go/providers/tokenizer"
)

// Message roles.
const (
	RoleAssistant = providers.RoleAssistant
	RoleSystem    = providers.RoleSystem
	RoleTool      = providers.RoleTool
	RoleUser      = providers.RoleUser
)

// Finish reasons.
const (
	FinishReasonContentFilter = providers.FinishReasonContentFilter
	FinishReasonLength        = providers.FinishReasonLength
	FinishReasonStop          = providers.FinishReasonStop
	FinishReasonToolCalls     = providers.FinishReasonToolCalls
)

// Cache control types.
const (
	CacheControlTypeEphemeral = providers.CacheControlTypeEphemeral
)

// Cache control TTL values.
const (
	CacheControlTTL5m = providers.CacheControlTTL5m
	CacheControlTTL1h = providers.CacheControlTTL1h
)

// ReasoningEffort levels.
const (
	ReasoningEffortAuto   = providers.ReasoningEffortAuto
	ReasoningEffortHigh   = providers.ReasoningEffortHigh
	ReasoningEffortLow    = providers.ReasoningEffortLow
	ReasoningEffortMedium = providers.ReasoningEffortMedium
	ReasoningEffortNone   = providers.ReasoningEffortNone
)

// Content types for ContentPart.
const (
	ContentTypeImageURL   = providers.ContentTypeImageURL
	ContentTypeInputAudio = providers.ContentTypeInputAudio
	ContentTypeText       = providers.ContentTypeText
	ContentTypeVideoURL   = providers.ContentTypeVideoURL
)

// Provider types.
type (
	Capabilities       = providers.Capabilities
	CapabilityProvider = providers.CapabilityProvider
	ChatBuilder        = providers.ChatBuilder
	ChatModel          = providers.ChatModel
	EmbeddingProvider  = providers.EmbeddingProvider
	ModelCapabilities  = providers.ModelCapabilities
	ModelLister        = providers.ModelLister
	ModelOption        = providers.ModelOption
	Provider           = providers.Provider
)

// Encoding for token estimation.
type Encoding = tokenizer.Encoding

// Tokenizer encoding constants.
const (
	EncodingCl100kBase = tokenizer.Cl100kBase
	EncodingClaude     = tokenizer.Claude
	EncodingGemini     = tokenizer.Gemini
	EncodingO200kBase  = tokenizer.O200kBase
	EncodingP50kBase   = tokenizer.P50kBase
	EncodingP50kEdit   = tokenizer.P50kEdit
	EncodingR50kBase   = tokenizer.R50kBase
)

// Request/Response types.
type (
	CacheControlParam   = providers.CacheControlParam
	CacheControlTTL     = providers.CacheControlTTL
	CacheControlType    = providers.CacheControlType
	ChatCompletion      = providers.ChatCompletion
	ChatCompletionChunk = providers.ChatCompletionChunk
	Choice              = providers.Choice
	ChunkChoice         = providers.ChunkChoice
	ChunkDelta          = providers.ChunkDelta
	CompletionParams    = providers.CompletionParams
	EmbeddingParams     = providers.EmbeddingParams
	EmbeddingResponse   = providers.EmbeddingResponse
	ModelsResponse      = providers.ModelsResponse
)

// Message types.
type (
	ContentPart = providers.ContentPart
	ImageURL    = providers.ImageURL
	InputAudio  = providers.InputAudio
	Message     = providers.Message
	Reasoning   = providers.Reasoning
	VideoURL    = providers.VideoURL
)

// Tool types.
type (
	Function           = providers.Function
	FunctionCall       = providers.FunctionCall
	Tool               = providers.Tool
	ToolCall           = providers.ToolCall
	ToolChoice         = providers.ToolChoice
	ToolChoiceFunction = providers.ToolChoiceFunction
)

// Response format types.
type (
	JSONSchema     = providers.JSONSchema
	ResponseFormat = providers.ResponseFormat
	StreamOptions  = providers.StreamOptions
)

// Usage and model types.
type (
	CacheCreation   = providers.CacheCreation
	EmbeddingData   = providers.EmbeddingData
	EmbeddingUsage  = providers.EmbeddingUsage
	Model           = providers.Model
	ReasoningEffort = providers.ReasoningEffort
	Usage           = providers.Usage
)

// Param types.
type (
	IntOpt    = param.Opt[int]
	FloatOpt  = param.Opt[float64]
	BoolOpt   = param.Opt[bool]
	StringOpt = param.Opt[string]
)

// Config types.
type (
	Config = config.Config
	Field  = config.Field
	Logger = config.Logger
	Option = config.Option
)

// Configuration options.
var (
	NewConfig      = config.New
	WithAPIKey     = config.WithAPIKey
	WithBaseURL    = config.WithBaseURL
	WithExtra      = config.WithExtra
	WithHTTPClient = config.WithHTTPClient
	WithLogger     = config.WithLogger
	WithTimeout    = config.WithTimeout
	WithUserID     = config.WithUserID
)

// Model construction and configuration.
var (
	NewChatModel = providers.NewChatModel
)

// Model capability options.
var (
	WithModelAudio     = providers.WithModelAudio
	WithModelImage     = providers.WithModelImage
	WithModelPDF       = providers.WithModelPDF
	WithModelReasoning = providers.WithModelReasoning
	WithModelStreaming = providers.WithModelStreaming
	WithModelTools     = providers.WithModelTools
	WithModelVideo     = providers.WithModelVideo
)

// Param convenience constructors.
var (
	OptInt    = param.Int
	OptFloat  = param.Float
	OptBool   = param.Bool
	OptString = param.String
)

// Sentinel errors for type checking with errors.Is().
var (
	ErrAuthentication      = errors.ErrAuthentication
	ErrContentFilter       = errors.ErrContentFilter
	ErrContextLength       = errors.ErrContextLength
	ErrInvalidRequest      = errors.ErrInvalidRequest
	ErrMissingAPIKey       = errors.ErrMissingAPIKey
	ErrModelNotFound       = errors.ErrModelNotFound
	ErrProvider            = errors.ErrProvider
	ErrRateLimit           = errors.ErrRateLimit
	ErrUnsupportedParam    = errors.ErrUnsupportedParam
	ErrUnsupportedProvider = errors.ErrUnsupportedProvider
)

// Error types.
type (
	AuthenticationError      = errors.AuthenticationError
	BaseError                = errors.BaseError
	ContentFilterError       = errors.ContentFilterError
	ContextLengthError       = errors.ContextLengthError
	InvalidRequestError      = errors.InvalidRequestError
	MissingAPIKeyError       = errors.MissingAPIKeyError
	ModelNotFoundError       = errors.ModelNotFoundError
	ProviderError            = errors.ProviderError
	RateLimitError           = errors.RateLimitError
	UnsupportedParamError    = errors.UnsupportedParamError
	UnsupportedProviderError = errors.UnsupportedProviderError
)

// Protocol types — Anthropic.
type (
	AnthropicMessageRequest  = anthropic.MessageRequest
	AnthropicMessageResponse = anthropic.MessageResponse
	AnthropicContentBlock    = anthropic.ContentBlock
	AnthropicStreamEvent     = anthropic.StreamEvent
)

// Protocol converters — Anthropic.
var (
	AnthropicToCompletionParams = anthropic.ToCompletionParams
	AnthropicFromCompletion     = anthropic.FromCompletion
	NewAnthropicStreamAdapter   = anthropic.NewStreamAdapter
)

// Protocol types — Responses API.
type (
	ResponsesRequest     = responses.Request
	ResponsesResponse    = responses.Response
	ResponsesOutputItem  = responses.OutputItem
	ResponsesStreamEvent = responses.StreamEvent
)

// Protocol converters — Responses API.
var (
	ResponsesToCompletionParams = responses.ToCompletionParams
	ResponsesFromCompletion     = responses.FromCompletion
	NewResponsesStreamAdapter   = responses.NewStreamAdapter
)

// Fallback types.
type (
	Router         = fallback.Router
	AllFailedError = fallback.AllFailedError
	RetryPolicy    = fallback.RetryPolicy
	Selector       = fallback.Selector
)

// Fallback router construction.
var (
	NewRouter                        = fallback.New
	NewDefaultRetryPolicy            = fallback.NewDefaultRetryPolicy
	NewRandomSelector                = fallback.NewRandomSelector
	NewRoundRobinSelector            = fallback.NewRoundRobinSelector
	WithRouterSelector               = fallback.WithSelector
	WithRouterRetryPolicy            = fallback.WithRetryPolicy
	WithRouterMaxAttemptsPerProvider = fallback.WithMaxAttemptsPerProvider
)

// Token estimation functions.
var (
	CountTokens             = tokenizer.CountTokens
	CountTokensWithEncoding = tokenizer.CountTokensWithEncoding
	CountText               = tokenizer.CountText
)
