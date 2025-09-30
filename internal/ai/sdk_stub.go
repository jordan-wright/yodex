package ai

// This stub keeps the OpenAI Go SDK v3 as a retained dependency so
// `go mod tidy` does not remove it before we wire integration.
// The actual usage will be implemented in subsequent tasks.

import (
	openai "github.com/openai/openai-go/v3"
)

// Reference a type to avoid unused import removal.
var _ any = (*openai.Client)(nil)
