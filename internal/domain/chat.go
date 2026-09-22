package domain

type Message struct {
	Role    string // "system", "user", "assistant", "tool"
	Content string
}

type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature *float32
	Stream      bool
}

type ChatResult struct {
	Answer           string
	PromptTokens     int32
	CompletionTokens int32
	TotalTokens      int32
	Provider         string
	FinishReason     string
}

type ChatChunk struct {
	ContentDelta     string
	// PromptTokens     int32
	// CompletionTokens int32
	// TotalTokens      int32
	// Provider         string
	FinishReason     string
}
