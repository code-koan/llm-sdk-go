# Token Estimation

Token estimation for pre-request quota holding and context-length checking. Zero network calls.

## Strategies

| Strategy | Models | Accuracy |
|----------|--------|----------|
| tiktoken (BPE) | OpenAI (GPT-4o, GPT-4, o1, o3, o4) | Exact |
| Character heuristic | Claude, Gemini, unknown | ~80-90% |

## Usage

### Auto-Detect by Model Name

```go
import (
    llmsdk "github.com/code-koan/llm-sdk-go"
)

messages := []llmsdk.Message{
    {Role: llmsdk.RoleUser, Content: "Hello, how are you?"},
}

count, err := llmsdk.CountTokens(messages, "gpt-4o")
```

### Explicit Encoding (User-Defined Models)

```go
count, err := llmsdk.CountTokensWithEncoding(messages, llmsdk.EncodingClaude)
```

### Raw Text

```go
count, err := llmsdk.CountText("Hello world", "gpt-4o")
```

## Encoding Constants

| Constant | Strategy | Models |
|----------|----------|--------|
| EncodingO200kBase | tiktoken | GPT-4o, o1, o3, o4, chatgpt |
| EncodingCl100kBase | tiktoken | GPT-4, GPT-3.5, text-embedding |
| EncodingClaude | Heuristic | Claude (all versions) |
| EncodingGemini | Heuristic | Gemini (all versions) |
| EncodingP50kBase | tiktoken | davinci-002, davinci-003 |
| EncodingP50kEdit | tiktoken | Edit models |
| EncodingR50kBase | tiktoken | GPT-3 legacy |

## Multimodal

Images, audio, and video are counted with fixed token estimates:

| Media | Detail | Tokens |
|-------|--------|--------|
| Image | low | 85 |
| Image | high / auto | 765 (4 tiles x 170 + 85) |
| Audio | — | 256 |
| Video | — | 8192 |

Image estimates assume typical 1024x1024 dimensions. For precise image token counts, use the provider API directly.
