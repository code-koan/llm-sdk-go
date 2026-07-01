package main

import (
	"context"
	"fmt"
	"os"

	llmsdk "github.com/code-koan/llm-sdk-go"
	"github.com/code-koan/llm-sdk-go/providers/openai"
)

func main() {
	// Step 1: Create a ChatModel with capability configuration
	m, err := openai.NewChatModel("gpt-4o-mini",
		llmsdk.WithModelTools(),
		llmsdk.WithModelStreaming(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create model: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Query capabilities
	caps := m.Capabilities()
	fmt.Printf("Model: %s\n", m.ModelID())
	fmt.Printf("  Tools:     %v\n", caps.Tools)
	fmt.Printf("  Streaming: %v\n", caps.Streaming)
	fmt.Printf("  Audio:     %v\n", caps.Audio)
	fmt.Printf("  Image:     %v\n", caps.Image)

	// Step 3: Build a chat request with method chaining
	params := m.NewChat().
		WithSystem("You are a helpful assistant.").
		WithText("Hello, how are you?").
		WithMaxTokens(100).
		Build()

	fmt.Printf("\nBuilt CompletionParams:\n")
	fmt.Printf("  Model:     %s\n", params.Model)
	fmt.Printf("  Messages:  %d\n", len(params.Messages))
	fmt.Printf("  MaxTokens: %d\n", *params.MaxTokens)

	// Step 4: Execute directly
	ctx := context.Background()
	resp, err := m.NewChat().
		WithText("Say 'hello world' in JSON format.").
		WithMaxTokens(50).
		Exec(ctx)

	if err != nil {
		fmt.Fprintf(os.Stderr, "exec: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nResponse:\n")
	if len(resp.Choices) > 0 {
		fmt.Printf("  %s\n", resp.Choices[0].Message.ContentString())
	}
	fmt.Printf("  Usage: %d prompt, %d completion, %d total\n",
		resp.Usage.PromptTokens,
		resp.Usage.CompletionTokens,
		resp.Usage.TotalTokens,
	)
}
