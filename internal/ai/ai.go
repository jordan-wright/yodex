package ai

import (
	"context"
	"io"
)

// TextClient generates text and structured JSON using a text model.
type TextClient interface {
	GenerateText(ctx context.Context, model, system, prompt string) (string, error)
	GenerateTextWithUsage(ctx context.Context, model, system, prompt string) (string, TokenUsage, error)
}

// TTSClient synthesizes speech audio from text.
type TTSClient interface {
	TTS(ctx context.Context, model, voice, text string, w io.Writer) error
}
