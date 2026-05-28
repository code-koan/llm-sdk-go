// Gateway — thin HTTP API that wraps llm-sdk-go providers.
//
//	PORT=8080 LLM_PROVIDER=openai go run ./examples/gateway
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"

	llmsdk "github.com/code-koan/llm-sdk-go"
	"github.com/code-koan/llm-sdk-go/providers/anthropic"
	"github.com/code-koan/llm-sdk-go/providers/ollama"
	"github.com/code-koan/llm-sdk-go/providers/openai"
)

func main() {
	provider, err := newProvider()
	if err != nil {
		log.Fatalf("gateway: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", handleChat(provider))
	mux.HandleFunc("GET /v1/models", handleModels(provider))

	addr := ":" + envOrDefault("PORT", "8080")
	log.Printf("gateway: listening on %s (provider: %s)", addr, provider.Name())
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

func handleChat(p llmsdk.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var params llmsdk.CompletionParams
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
			return
		}

		if params.Stream {
			handleStream(w, r, p, params)
			return
		}

		resp, err := p.Completion(r.Context(), params)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func handleStream(w http.ResponseWriter, r *http.Request, p llmsdk.Provider, params llmsdk.CompletionParams) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	ch, errCh := p.CompletionStream(r.Context(), params)

	for {
		select {
		case chunk, ok := <-ch:
			if !ok {
				_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}
			data, _ := json.Marshal(chunk)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case err := <-errCh:
			if err != nil {
				data, _ := json.Marshal(map[string]string{"error": err.Error()})
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func handleModels(p llmsdk.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lister, ok := p.(llmsdk.ModelLister)
		if !ok {
			writeJSON(w, http.StatusOK, llmsdk.ModelsResponse{Object: "list", Data: nil})
			return
		}
		resp, err := lister.ListModels(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func newProvider() (llmsdk.Provider, error) {
	provider := envOrDefault("LLM_PROVIDER", "openai")
	valid := []string{"openai", "anthropic", "ollama"}
	if !slices.Contains(valid, provider) {
		return nil, fmt.Errorf("unknown provider %q (valid: %v)", provider, valid)
	}

	opts := []llmsdk.Option{llmsdk.WithAPIKey(os.Getenv("LLM_API_KEY"))}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		opts = append(opts, llmsdk.WithBaseURL(v))
	}

	switch provider {
	case "anthropic":
		return anthropic.New(opts...)
	case "ollama":
		return ollama.New(opts...)
	default:
		return openai.New(opts...)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
