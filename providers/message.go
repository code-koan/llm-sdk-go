package providers

// System creates a system message.
func System(content string) Message {
	return Message{Role: RoleSystem, Content: content}
}

// User creates a user message. Content can be a string or []ContentPart.
func User(content any) Message {
	return Message{Role: RoleUser, Content: content}
}

// Assistant creates an assistant message. Content can be a string or []ContentPart.
func Assistant(content any) Message {
	return Message{Role: RoleAssistant, Content: content}
}

// ToolResult creates a tool result message.
func ToolResult(callID, content string) Message {
	return Message{Role: RoleTool, ToolCallID: callID, Content: content}
}

// Text creates a text content part.
func Text(text string) ContentPart {
	return ContentPart{Type: ContentTypeText, Text: text}
}

// Image creates an image URL content part.
func Image(url string) ContentPart {
	return ContentPart{Type: ContentTypeImageURL, ImageURL: &ImageURL{URL: url}}
}

// Audio creates an audio input content part. data is base64-encoded or data URL.
func Audio(data, format string) ContentPart {
	return ContentPart{Type: ContentTypeInputAudio, InputAudio: &InputAudio{Data: data, Format: format}}
}

// Video creates a video URL content part.
func Video(url string) ContentPart {
	return ContentPart{Type: ContentTypeVideoURL, VideoURL: &VideoURL{URL: url}}
}
