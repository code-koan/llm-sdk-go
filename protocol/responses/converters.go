package responses

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/code-koan/llm-sdk-go/providers"
)

// ToCompletionParams converts a Responses API request to SDK CompletionParams.
func ToCompletionParams(req *Request) (*providers.CompletionParams, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	messages := convertInputToMessages(req.Input, req.Instructions)
	if len(messages) == 0 {
		return nil, fmt.Errorf("input is required")
	}

	params := &providers.CompletionParams{
		Model:    req.Model,
		Messages: messages,
		Stream:   req.Stream,
	}

	if req.MaxOutputTokens > 0 {
		params.MaxTokens = &req.MaxOutputTokens
	}
	if req.Temperature != 0 {
		params.Temperature = &req.Temperature
	}
	if req.TopP != 0 {
		params.TopP = &req.TopP
	}
	if len(req.Tools) > 0 {
		params.Tools = convertTools(req.Tools)
	}
	if req.ToolChoice != nil {
		params.ToolChoice = req.ToolChoice
	}

	return params, nil
}

// FromCompletion converts an SDK ChatCompletion to a Responses API response.
func FromCompletion(completion *providers.ChatCompletion, req *Request) *Response {
	resp := &Response{
		ID:        fmt.Sprintf("resp_%d", time.Now().UnixNano()),
		Object:    "response",
		CreatedAt: time.Now().Unix(),
		Model:     req.Model,
		Status:    StatusCompleted,
	}

	if completion != nil && len(completion.Choices) > 0 {
		choice := completion.Choices[0]

		// Build output items.
		var outputItems []OutputItem

		// Text message.
		text := messageText(choice.Message.Content)
		if text != "" {
			outputItems = append(outputItems, OutputItem{
				Type: OutputItemTypeMessage,
				Role: "assistant",
				Content: []ContentPart{{
					Type: ContentTypeOutputText,
					Text: text,
				}},
			})
		}

		// Function calls.
		for _, tc := range choice.Message.ToolCalls {
			outputItems = append(outputItems, OutputItem{
				Type:      OutputItemTypeFunctionCall,
				ID:        tc.ID,
				CallID:    tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		if len(outputItems) == 0 {
			outputItems = append(outputItems, OutputItem{
				Type: OutputItemTypeMessage,
				Role: "assistant",
				Content: []ContentPart{{
					Type: ContentTypeOutputText,
					Text: "",
				}},
			})
		}
		resp.Output = outputItems

		if completion.Usage != nil {
			resp.Usage = &Usage{
				InputTokens:  completion.Usage.PromptTokens,
				OutputTokens: completion.Usage.CompletionTokens,
				TotalTokens:  completion.Usage.TotalTokens,
			}
		}
	}

	if resp.Usage == nil {
		resp.Usage = &Usage{}
	}

	return resp
}

// --- Internal converters ---

// convertInputToMessages converts Responses API input to SDK messages.
func convertInputToMessages(input any, instructions string) []providers.Message {
	messages := make([]providers.Message, 0, 4)
	if instructions != "" {
		messages = append(messages, providers.Message{Role: providers.RoleSystem, Content: instructions})
	}
	switch v := input.(type) {
	case string:
		messages = append(messages, providers.Message{Role: providers.RoleUser, Content: v})
	case []any:
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			switch m["type"] {
			case OutputItemTypeFunctionCallOutput:
				messages = append(messages, convertFunctionCallOutput(m))
			case OutputItemTypeFunctionCall:
				messages = append(messages, convertFunctionCall(m))
			default:
				messages = append(messages, convertDefaultMessage(m))
			}
		}
	}
	return messages
}

// convertFunctionCallOutput converts a function_call_output item to a tool message.
func convertFunctionCallOutput(item map[string]any) providers.Message {
	msg := providers.Message{Role: providers.RoleTool}
	if callID, ok := item["call_id"].(string); ok {
		msg.ToolCallID = callID
	}
	if output, ok := item["output"].(string); ok {
		msg.Content = output
	}
	return msg
}

// convertFunctionCall converts a function_call item to an assistant message with tool calls.
func convertFunctionCall(item map[string]any) providers.Message {
	msg := providers.Message{Role: providers.RoleAssistant, Content: ""}
	name, _ := item["name"].(string)
	callID, _ := item["call_id"].(string)
	args, _ := item["arguments"].(string)
	if name != "" && callID != "" {
		msg.ToolCalls = []providers.ToolCall{{
			ID:   callID,
			Type: "function",
			Function: providers.FunctionCall{
				Name:      name,
				Arguments: args,
			},
		}}
	}
	return msg
}

// convertDefaultMessage converts a generic item to a message.
func convertDefaultMessage(item map[string]any) providers.Message {
	role, _ := item["role"].(string)
	content := item["content"]
	return providers.Message{Role: role, Content: content}
}

// convertTools converts Responses API tools to SDK tools.
func convertTools(tools []Tool) []providers.Tool {
	result := make([]providers.Tool, 0, len(tools))
	for _, t := range tools {
		sdkTool := providers.Tool{
			Type: "function",
			Function: providers.Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
		result = append(result, sdkTool)
	}
	return result
}

// messageText extracts string content from a message.
func messageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}
