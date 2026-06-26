package seedance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/code-koan/llm-sdk-go/config"
	"github.com/code-koan/llm-sdk-go/errors"
	"github.com/code-koan/llm-sdk-go/providers"
)

const (
	testAPIKey     = "test-api-key"
	testModel      = "doubao-seedance-2-0-260128"
	contentGenPath = "/contents/generations/tasks"
)

// newTestServer creates a test HTTP server with the given handler.
func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// createTestProvider creates a Seedance provider pointing to the given mock server.
func createTestProvider(t *testing.T, ts *httptest.Server) *Provider {
	t.Helper()
	provider, err := New(
		config.WithAPIKey(testAPIKey),
		config.WithBaseURL(ts.URL),
	)
	require.NoError(t, err)
	require.NotNil(t, provider)
	return provider
}

// readBody reads and unmarshals the request body into the given value.
func readBody(t *testing.T, r *http.Request, v any) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	defer r.Body.Close()
	require.NoError(t, json.Unmarshal(body, v))
}

// ---------------------------------------------------------------------------
// New
// ---------------------------------------------------------------------------

func TestNew_Defaults(t *testing.T) {
	t.Setenv("ARK_API_KEY", "env-api-key")

	provider, err := New()
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Equal(t, "seedance", provider.Name())
}

func TestNew_MissingAPIKey(t *testing.T) {
	t.Setenv("ARK_API_KEY", "")

	provider, err := New()
	require.Nil(t, provider)
	require.Error(t, err)

	var missingKeyErr *errors.MissingAPIKeyError
	require.ErrorAs(t, err, &missingKeyErr)
	require.Equal(t, "seedance", missingKeyErr.Provider)
	require.Equal(t, "ARK_API_KEY", missingKeyErr.EnvVar)
}

func TestNew_WithOptions(t *testing.T) {
	t.Parallel()

	provider, err := New(
		config.WithAPIKey("custom-key"),
		config.WithBaseURL("http://custom.base.url"),
	)
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.Equal(t, "seedance", provider.Name())
}

// ---------------------------------------------------------------------------
// SubmitTask
// ---------------------------------------------------------------------------

func TestSubmitTask_MissingModel(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	_, err := provider.SubmitTask(context.Background(), providers.AsyncTaskParams{
		Model:   "",
		Content: "test prompt",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "model is required")
}

func TestSubmitTask_TextToVideo(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, contentGenPath, r.URL.Path)
		require.Equal(t, "Bearer "+testAPIKey, r.Header.Get("Authorization"))

		var body map[string]any
		readBody(t, r, &body)
		require.Equal(t, testModel, body["model"])
		require.NotEmpty(t, body["content"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"cgt-20251125163544-qrj4f"}`))
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	task, err := provider.SubmitTask(context.Background(), providers.AsyncTaskParams{
		Model:   testModel,
		Content: "写实风格，晴朗的蓝天之下，一大片白色的雏菊花田",
	})
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, "cgt-20251125163544-qrj4f", task.ID)
	require.Equal(t, providers.AsyncTaskQueued, task.Status)
}

func TestSubmitTask_ImageToVideoFirstFrame(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]any
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, contentGenPath, r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		require.NoError(t, json.Unmarshal(body, &capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"test-image-to-video-id"}`))
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	_, err := provider.SubmitTask(context.Background(), providers.AsyncTaskParams{
		Model: testModel,
		Content: []providers.ContentPart{
			{
				Type: "image_url",
				ImageURL: &providers.ImageURL{
					URL:  "https://example.com/first_frame.png",
					Role: "first_frame",
				},
			},
			{
				Type: "text",
				Text: "让画面中的花朵随风摇曳",
			},
		},
	})
	require.NoError(t, err)

	// Verify role field is present in the request body.
	content := capturedBody["content"].([]any)
	require.Len(t, content, 2)

	firstItem := content[0].(map[string]any)
	require.Equal(t, "image_url", firstItem["type"])
	require.Equal(t, "first_frame", firstItem["role"])

	imgURL := firstItem["image_url"].(map[string]any)
	require.Equal(t, "https://example.com/first_frame.png", imgURL["url"])

	secondItem := content[1].(map[string]any)
	require.Equal(t, "text", secondItem["type"])
	require.Equal(t, "让画面中的花朵随风摇曳", secondItem["text"])
}

func TestSubmitTask_ExtraParams(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]any
	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, contentGenPath, r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		defer r.Body.Close()

		require.NoError(t, json.Unmarshal(body, &capturedBody))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"test-extra-id"}`))
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	_, err := provider.SubmitTask(context.Background(), providers.AsyncTaskParams{
		Model:   testModel,
		Content: "test prompt",
		Extra: map[string]any{
			"resolution": "720p",
			"duration":   5,
			"ratio":      "16:9",
			"watermark":  false,
		},
	})
	require.NoError(t, err)

	require.Equal(t, "720p", capturedBody["resolution"])
	require.Equal(t, float64(5), capturedBody["duration"])
	require.Equal(t, "16:9", capturedBody["ratio"])
	require.Equal(t, false, capturedBody["watermark"])
}

// ---------------------------------------------------------------------------
// GetTask
// ---------------------------------------------------------------------------

func TestGetTask_MissingTaskID(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("unexpected request")
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	_, err := provider.GetTask(context.Background(), "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "taskID is required")
}

func TestGetTask_Succeeded(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, contentGenPath+"/cgt-20251119202422-jcfm2", r.URL.Path)
		require.Equal(t, "Bearer "+testAPIKey, r.Header.Get("Authorization"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "cgt-20251119202422-jcfm2",
			"model": "doubao-seedance-2-0-260128",
			"status": "succeeded",
			"content": {
				"video_url": "https://ark-content-generation-cn-beijing.tos-cn-beijing.volces.com/xxx.mp4"
			},
			"usage": { "completion_tokens": 295800 },
			"duration": 5,
			"resolution": "1080p",
			"ratio": "9:16"
		}`))
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	task, err := provider.GetTask(context.Background(), "cgt-20251119202422-jcfm2")
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, providers.AsyncTaskSucceeded, task.Status)
	require.Equal(t, "cgt-20251119202422-jcfm2", task.ID)
	require.Equal(t, "doubao-seedance-2-0-260128", task.Model)
	require.Equal(t, "https://ark-content-generation-cn-beijing.tos-cn-beijing.volces.com/xxx.mp4", task.ResultURL)

	// Usage
	require.NotNil(t, task.Usage)
	require.Equal(t, 295800, task.Usage.CompletionTokens)
	require.Equal(t, 0, task.Usage.PromptTokens)
	require.Equal(t, 0, task.Usage.TotalTokens)

	// Extra metadata
	require.NotNil(t, task.Extra)
	require.Equal(t, "1080p", task.Extra["resolution"])
	require.Equal(t, "9:16", task.Extra["ratio"])
	require.Equal(t, int64(5), task.Extra["duration"])
}

func TestGetTask_Failed(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, contentGenPath+"/cgt-xxx", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "cgt-xxx",
			"status": "failed",
			"error": {
				"code": "InvalidParameter",
				"message": "Invalid parameter: duration"
			}
		}`))
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	task, err := provider.GetTask(context.Background(), "cgt-xxx")
	require.NoError(t, err)
	require.NotNil(t, task)
	require.Equal(t, providers.AsyncTaskFailed, task.Status)
	require.Equal(t, "cgt-xxx", task.ID)
	require.NotNil(t, task.Error)
	require.Equal(t, "InvalidParameter", task.Error.Code)
	require.Equal(t, "Invalid parameter: duration", task.Error.Message)
}

func TestGetTask_StatusMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		apiStatus      string
		expectedStatus providers.AsyncTaskStatus
	}{
		{name: "queued", apiStatus: "queued", expectedStatus: providers.AsyncTaskQueued},
		{name: "running", apiStatus: "running", expectedStatus: providers.AsyncTaskRunning},
		{name: "succeeded", apiStatus: "succeeded", expectedStatus: providers.AsyncTaskSucceeded},
		{name: "failed", apiStatus: "failed", expectedStatus: providers.AsyncTaskFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodGet, r.Method)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				resp := fmt.Sprintf(`{"id":"task-1","status":"%s"}`, tc.apiStatus)
				_, _ = w.Write([]byte(resp))
			})
			defer ts.Close()

			provider := createTestProvider(t, ts)

			task, err := provider.GetTask(context.Background(), "task-1")
			require.NoError(t, err)
			require.NotNil(t, task)
			require.Equal(t, tc.expectedStatus, task.Status)
		})
	}
}

// ---------------------------------------------------------------------------
// Context cancellation
// ---------------------------------------------------------------------------

func TestSubmitTask_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ts := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		// This should never be reached because the context is already
		// cancelled before the HTTP client attempts a connection.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"unreachable"}`))
	})
	defer ts.Close()

	provider := createTestProvider(t, ts)

	_, err := provider.SubmitTask(ctx, providers.AsyncTaskParams{
		Model:   testModel,
		Content: "test prompt",
	})
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}
