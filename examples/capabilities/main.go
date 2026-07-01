// Example: Checking provider capabilities
//
// This example demonstrates how to query a provider's capabilities
// via the CapabilityProvider interface.
//
// Run with:
//
//	export OPENAI_API_KEY="sk-..."
//	go run main.go
package main

import (
	"fmt"
	"log"

	llmsdk "github.com/code-koan/llm-sdk-go"
	"github.com/code-koan/llm-sdk-go/providers"
	"github.com/code-koan/llm-sdk-go/providers/openai"
)

func main() {
	p, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// All built-in providers implement CapabilityProvider.
	var cp providers.CapabilityProvider = p
	caps := cp.Capabilities()

	fmt.Println("OpenAI capabilities:")
	fmt.Printf("  Completion:     %v\n", caps.Completion)
	fmt.Printf("  Streaming:      %v\n", caps.CompletionStreaming)
	fmt.Printf("  Tools:          %v\n", caps.CompletionTools)
	fmt.Printf("  Reasoning:      %v\n", caps.CompletionReasoning)
	fmt.Printf("  Image input:    %v\n", caps.CompletionImage)
	fmt.Printf("  Audio input:    %v\n", caps.CompletionAudio)
	fmt.Printf("  Video input:    %v\n", caps.CompletionVideo)
	fmt.Printf("  Embeddings:     %v\n", caps.Embedding)
	fmt.Printf("  List models:    %v\n", caps.ListModels)
	fmt.Printf("  TTS:            %v\n", caps.TTS)
	fmt.Printf("  STT:            %v\n", caps.STT)
	fmt.Printf("  Async gen:      %v\n", caps.AsyncGeneration)

	// The root llmsdk package also re-exports Capabilities.
	_ = llmsdk.Capabilities{}
}
