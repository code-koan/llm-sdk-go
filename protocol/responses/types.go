package responses

import "github.com/code-koan/llm-sdk-go/providers"

// Response status constants.
const (
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Output item type constants.
const (
	OutputItemTypeMessage            = "message"
	OutputItemTypeFunctionCall       = "function_call"
	OutputItemTypeFunctionCallOutput = "function_call_output"
)

// Content part type constants.
const (
	ContentTypeOutputText = "output_text"
	ContentTypeRefusal    = "refusal"
)

// --- Request types ---

// Request represents an OpenAI Responses API request body.
type Request struct {
	Model           string  `json:"model"`
	Input           any     `json:"input,omitempty"` // string | []InputItem
	Instructions    string  `json:"instructions,omitempty"`
	MaxOutputTokens int     `json:"max_output_tokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
	TopP            float64 `json:"top_p,omitempty"`
	Tools           []Tool  `json:"tools,omitempty"`
	ToolChoice      any     `json:"tool_choice,omitempty"`
	Stream          bool    `json:"stream,omitempty"`
}

// Tool in Responses API format.
type Tool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// --- Response types ---

// Response represents an OpenAI Responses API response.
type Response struct {
	ID        string       `json:"id"`
	Object    string       `json:"object"`
	CreatedAt int64        `json:"created_at"`
	Model     string       `json:"model"`
	Status    string       `json:"status,omitempty"`
	Output    []OutputItem `json:"output"`
	Usage     *Usage       `json:"usage,omitempty"`
}

// OutputItem represents an item in the response output.
type OutputItem struct {
	Type      string        `json:"type"`
	ID        string        `json:"id,omitempty"`
	Role      string        `json:"role,omitempty"`
	Content   []ContentPart `json:"content,omitempty"`
	Status    string        `json:"status,omitempty"`
	Name      string        `json:"name,omitempty"`
	Arguments string        `json:"arguments,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Output    string        `json:"output,omitempty"`
}

// ContentPart represents a content part in a response output item.
type ContentPart struct {
	Type        string       `json:"type"`
	Text        string       `json:"text,omitempty"`
	Refusal     string       `json:"refusal,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

// Annotation represents content annotations (e.g., URL citations).
type Annotation struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
	Text string `json:"text,omitempty"`
}

// Usage in Responses API format.
type Usage struct {
	InputTokens     int `json:"input_tokens"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// ToSDK converts Usage to the SDK's providers.Usage format.
func (u Usage) ToSDK() *providers.Usage {
	return &providers.Usage{
		PromptTokens:     u.InputTokens,
		CompletionTokens: u.OutputTokens,
		TotalTokens:      u.TotalTokens,
		ReasoningTokens:  u.ReasoningTokens,
	}
}
