package client

// ---------------------------------------------------------------------------
// AI Chat
// ---------------------------------------------------------------------------

// AIChatRequest is the request body for POST /ai/chat
type AIChatRequest struct {
	Messages []AIChatMessage `json:"messages"`
	Model    string          `json:"model,omitempty"`   // "haiku" or "sonnet"
	Context  string          `json:"context,omitempty"` // piped stdin content
	Stream   bool            `json:"stream"`
}

// AIChatMessage represents a single message in the conversation.
type AIChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// AIChatStreamEvent is a single event from the SSE stream.
type AIChatStreamEvent struct {
	Type  string            `json:"type"`            // content_delta, content_stop, message_start, error, done
	Text  string            `json:"text,omitempty"`
	Error string            `json:"error,omitempty"`
	Usage *AIChatTokenUsage `json:"usage,omitempty"` // present on "done" event
	Meta  *ResponseMeta     `json:"meta,omitempty"`  // present on "done" event
}

// AIChatTokenUsage contains token counts and credit cost from a completed AI interaction.
type AIChatTokenUsage struct {
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CreditsUsed  int    `json:"credits_used"`
	Model        string `json:"model"`
}
