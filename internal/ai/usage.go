package ai

import "github.com/openai/openai-go/v3/responses"

// TokenUsage captures token usage returned by the Responses API.
type TokenUsage struct {
	InputTokens     int64
	OutputTokens    int64
	TotalTokens     int64
	CachedTokens    int64
	ReasoningTokens int64
}

func (u TokenUsage) Add(other TokenUsage) TokenUsage {
	return TokenUsage{
		InputTokens:     u.InputTokens + other.InputTokens,
		OutputTokens:    u.OutputTokens + other.OutputTokens,
		TotalTokens:     u.TotalTokens + other.TotalTokens,
		CachedTokens:    u.CachedTokens + other.CachedTokens,
		ReasoningTokens: u.ReasoningTokens + other.ReasoningTokens,
	}
}

func usageFromResponse(usage responses.ResponseUsage) TokenUsage {
	return TokenUsage{
		InputTokens:     usage.InputTokens,
		OutputTokens:    usage.OutputTokens,
		TotalTokens:     usage.TotalTokens,
		CachedTokens:    usage.InputTokensDetails.CachedTokens,
		ReasoningTokens: usage.OutputTokensDetails.ReasoningTokens,
	}
}
