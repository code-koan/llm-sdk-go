// Package seedance provides a Seedance (video generation) provider for llm-sdk.
package seedance

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"github.com/code-koan/llm-sdk-go/config"
	"github.com/code-koan/llm-sdk-go/errors"
	"github.com/code-koan/llm-sdk-go/providers"
)

// Provider configuration constants.
const (
	defaultBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
	envAPIKey      = "ARK_API_KEY"
	providerName   = "seedance"
)

// Ensure Provider implements the required interfaces.
var (
	_ providers.AsyncTaskProvider  = (*Provider)(nil)
	_ providers.CapabilityProvider = (*Provider)(nil)
	_ providers.ErrorConverter     = (*Provider)(nil)
	_ providers.Provider           = (*Provider)(nil)
)

// Provider implements the providers.AsyncTaskProvider interface for
// Seedance video generation via the Volcengine ARK runtime API.
type Provider struct {
	client *arkruntime.Client
	config *config.Config
}

// New creates a new Seedance provider.
func New(opts ...config.Option) (*Provider, error) {
	cfg, err := config.New(opts...)
	if err != nil {
		return nil, err
	}

	apiKey := cfg.ResolveAPIKey(envAPIKey)
	if apiKey == "" {
		return nil, errors.NewMissingAPIKeyError(providerName, envAPIKey)
	}

	baseURL, err := cfg.ResolveBaseURL("", defaultBaseURL)
	if err != nil {
		return nil, fmt.Errorf("seedance: %w", err)
	}

	client := arkruntime.NewClientWithApiKey(
		apiKey,
		arkruntime.WithBaseUrl(baseURL),
	)

	return &Provider{client: client, config: cfg}, nil
}

// Name returns the provider name.
func (p *Provider) Name() string { return providerName }

// Completion is not supported for video generation.
func (p *Provider) Completion(
	ctx context.Context,
	params providers.CompletionParams,
) (*providers.ChatCompletion, error) {
	return nil, fmt.Errorf("seedance: Completion not supported, use SubmitTask for video generation")
}

// CompletionStream is not supported for video generation.
func (p *Provider) CompletionStream(
	ctx context.Context,
	params providers.CompletionParams,
) (<-chan providers.ChatCompletionChunk, <-chan error) {
	errCh := make(chan error, 1)
	errCh <- fmt.Errorf("seedance: CompletionStream not supported, use SubmitTask for video generation")
	close(errCh)
	return nil, errCh
}

// Capabilities returns the capabilities of the provider.
func (p *Provider) Capabilities() providers.Capabilities {
	return providers.Capabilities{
		AsyncGeneration: true,
	}
}

// SubmitTask creates an async video generation task.
func (p *Provider) SubmitTask(ctx context.Context, params providers.AsyncTaskParams) (*providers.AsyncTask, error) {
	if params.Model == "" {
		return nil, fmt.Errorf("seedance: model is required")
	}

	content, err := buildContentItems(params.Content)
	if err != nil {
		return nil, fmt.Errorf("seedance: %w", err)
	}

	req := model.CreateContentGenerationTaskRequest{
		Model:   params.Model,
		Content: content,
	}
	applyExtra(&req, params.Extra)

	resp, err := p.client.CreateContentGenerationTask(ctx, req)
	if err != nil {
		return nil, p.ConvertError(err)
	}

	return &providers.AsyncTask{
		ID:     resp.ID,
		Model:  params.Model,
		Status: providers.AsyncTaskQueued,
	}, nil
}

// GetTask retrieves an async generation task by ID.
func (p *Provider) GetTask(ctx context.Context, taskID string) (*providers.AsyncTask, error) {
	if taskID == "" {
		return nil, fmt.Errorf("seedance: taskID is required")
	}

	resp, err := p.client.GetContentGenerationTask(ctx, model.GetContentGenerationTaskRequest{ID: taskID})
	if err != nil {
		return nil, p.ConvertError(err)
	}

	task := &providers.AsyncTask{
		ID:     resp.ID,
		Model:  resp.Model,
		Status: convertTaskStatus(resp.Status),
	}

	if resp.Error != nil {
		task.Error = &providers.AsyncTaskError{
			Code:    resp.Error.Code,
			Message: resp.Error.Message,
		}
	}

	// Set result URL: prefer video_url, fall back to file_url, then last_frame_url.
	switch {
	case resp.Content.VideoURL != "":
		task.ResultURL = resp.Content.VideoURL
	case resp.Content.FileURL != "":
		task.ResultURL = resp.Content.FileURL
	case resp.Content.LastFrameURL != "":
		task.ResultURL = resp.Content.LastFrameURL
	}

	// Map usage when available.
	if resp.Usage.TotalTokens > 0 || resp.Usage.PromptTokens > 0 || resp.Usage.CompletionTokens > 0 {
		task.Usage = &providers.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	// Populate extra with response metadata.
	task.Extra = make(map[string]any)
	if resp.Resolution != nil {
		task.Extra["resolution"] = *resp.Resolution
	}
	if resp.Ratio != nil {
		task.Extra["ratio"] = *resp.Ratio
	}
	if resp.Duration != nil {
		task.Extra["duration"] = *resp.Duration
	}
	if resp.Frames != nil {
		task.Extra["frames"] = *resp.Frames
	}
	if resp.FramesPerSecond != nil {
		task.Extra["frames_per_second"] = *resp.FramesPerSecond
	}
	if resp.Seed != nil {
		task.Extra["seed"] = *resp.Seed
	}
	if resp.UpdatedAt > 0 {
		task.Extra["updated_at"] = resp.UpdatedAt
	}
	if resp.RevisedPrompt != nil {
		task.Extra["revised_prompt"] = *resp.RevisedPrompt
	}
	if resp.GenerateAudio != nil {
		task.Extra["generate_audio"] = *resp.GenerateAudio
	}
	if resp.SafetyIdentifier != nil {
		task.Extra["safety_identifier"] = *resp.SafetyIdentifier
	}
	if resp.Content.LastFrameURL != "" {
		task.Extra["last_frame_url"] = resp.Content.LastFrameURL
	}
	if resp.Content.FileURL != "" {
		task.Extra["file_url"] = resp.Content.FileURL
	}

	return task, nil
}

// ConvertError converts a provider SDK error to a normalized SDK error.
func (p *Provider) ConvertError(err error) error {
	var apiErr *model.APIError
	if stderrors.As(err, &apiErr) {
		return convertAPIError(apiErr)
	}

	var reqErr *model.RequestError
	if stderrors.As(err, &reqErr) {
		return convertRequestError(reqErr)
	}

	return errors.NewProviderError(providerName, err)
}

// buildContentItems converts the generic Content field to SDK content items.
func buildContentItems(content any) ([]*model.CreateContentGenerationContentItem, error) {
	switch c := content.(type) {
	case nil:
		return nil, fmt.Errorf("content is required")
	case string:
		return []*model.CreateContentGenerationContentItem{{
			Type: model.ContentGenerationContentItemTypeText,
			Text: &c,
		}}, nil
	case []providers.ContentPart:
		items := make([]*model.CreateContentGenerationContentItem, 0, len(c))
		for _, part := range c {
			item := &model.CreateContentGenerationContentItem{
				Type: model.ContentGenerationContentItemType(part.Type),
			}
			switch part.Type {
			case "text":
				item.Text = &part.Text
			case "image_url":
				if part.ImageURL != nil {
					item.ImageURL = &model.ImageURL{URL: part.ImageURL.URL}
					if part.ImageURL.Role != "" {
						item.Role = &part.ImageURL.Role
					}
				}
			default:
				return nil, fmt.Errorf("unsupported content type: %s", part.Type)
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported content type: %T", content)
	}
}

// applyExtra maps vendor-specific extra parameters to the SDK request.
func applyExtra(req *model.CreateContentGenerationTaskRequest, extra map[string]any) {
	for k, v := range extra {
		switch k {
		case "resolution":
			if s, ok := v.(string); ok {
				req.Resolution = &s
			}
		case "ratio":
			if s, ok := v.(string); ok {
				req.Ratio = &s
			}
		case "safety_identifier":
			if s, ok := v.(string); ok {
				req.SafetyIdentifier = &s
			}
		case "callback_url":
			if s, ok := v.(string); ok {
				req.CallbackUrl = &s
			}
		case "service_tier":
			if s, ok := v.(string); ok {
				req.ServiceTier = &s
			}
		case "return_last_frame":
			if b, ok := v.(bool); ok {
				req.ReturnLastFrame = &b
			}
		case "seed":
			if f, ok := toInt64(v); ok {
				req.Seed = &f
			}
		case "generate_audio":
			if b, ok := v.(bool); ok {
				req.GenerateAudio = &b
			}
		case "draft":
			if b, ok := v.(bool); ok {
				req.Draft = &b
			}
		case "camera_fixed":
			if b, ok := v.(bool); ok {
				req.CameraFixed = &b
			}
		case "watermark":
			if b, ok := v.(bool); ok {
				req.Watermark = &b
			}
		case "duration":
			if f, ok := toInt64(v); ok {
				req.Duration = &f
			}
		case "frames":
			if f, ok := toInt64(v); ok {
				req.Frames = &f
			}
		case "priority":
			if f, ok := toInt32(v); ok {
				req.Priority = &f
			}
		case "execution_expires_after":
			if f, ok := toInt64(v); ok {
				req.ExecutionExpiresAfter = &f
			}
		default:
			if req.ExtraBody == nil {
				req.ExtraBody = make(model.ExtraBody)
			}
			req.ExtraBody[k] = v
		}
	}
}

// toInt64 attempts to convert a value to int64, handling JSON number decoding.
func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int64:
		return val, true
	case float64:
		return int64(val), true
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	default:
		return 0, false
	}
}

// toInt32 attempts to convert a value to int32, handling JSON number decoding.
func toInt32(v any) (int32, bool) {
	switch val := v.(type) {
	case int32:
		return val, true
	case float64:
		return int32(val), true
	case int:
		return int32(val), true
	case int64:
		return int32(val), true
	default:
		return 0, false
	}
}

// convertTaskStatus maps SDK task status to SDK normalized status.
func convertTaskStatus(status string) providers.AsyncTaskStatus {
	switch status {
	case model.StatusSucceeded:
		return providers.AsyncTaskSucceeded
	case model.StatusFailed:
		return providers.AsyncTaskFailed
	case model.StatusRunning:
		return providers.AsyncTaskRunning
	case model.StatusQueued:
		return providers.AsyncTaskQueued
	case model.StatusCancelled:
		return providers.AsyncTaskFailed
	default:
		return providers.AsyncTaskRunning
	}
}

// convertAPIError maps a SDK APIError to a normalized SDK error.
func convertAPIError(apiErr *model.APIError) error {
	switch apiErr.HTTPStatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.NewAuthenticationError(providerName, apiErr)
	case http.StatusTooManyRequests:
		return errors.NewRateLimitError(providerName, apiErr)
	case http.StatusNotFound:
		return errors.NewModelNotFoundError(providerName, apiErr)
	case http.StatusBadRequest:
		return errors.NewInvalidRequestError(providerName, apiErr)
	default:
		if apiErr.HTTPStatusCode >= 500 {
			return errors.NewProviderError(providerName, apiErr)
		}
		return errors.NewInvalidRequestError(providerName, apiErr)
	}
}

// convertRequestError maps a SDK RequestError to a normalized SDK error.
func convertRequestError(reqErr *model.RequestError) error {
	switch reqErr.HTTPStatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return errors.NewAuthenticationError(providerName, reqErr)
	case http.StatusTooManyRequests:
		return errors.NewRateLimitError(providerName, reqErr)
	case http.StatusNotFound:
		return errors.NewModelNotFoundError(providerName, reqErr)
	case http.StatusBadRequest:
		return errors.NewInvalidRequestError(providerName, reqErr)
	default:
		if reqErr.HTTPStatusCode >= 500 {
			return errors.NewProviderError(providerName, reqErr)
		}
		return errors.NewInvalidRequestError(providerName, reqErr)
	}
}
