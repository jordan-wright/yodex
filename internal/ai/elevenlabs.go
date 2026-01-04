package ai

import (
	"context"
	"errors"
	"io"

	elevenlabs "github.com/agentplexus/go-elevenlabs"
)

// ElevenLabsClient wraps the ElevenLabs SDK for TTS.
type ElevenLabsClient struct {
	apiKey string
	sdk    *elevenlabs.Client
}

// NewElevenLabs constructs a new ElevenLabs client. The apiKey is required.
func NewElevenLabs(apiKey string) (*ElevenLabsClient, error) {
	if apiKey == "" {
		return nil, errors.New("ELEVENLABS_API_KEY is required")
	}
	sdk, err := elevenlabs.NewClient(elevenlabs.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &ElevenLabsClient{apiKey: apiKey, sdk: sdk}, nil
}

// TTS writes MP3 audio to the provided writer using the ElevenLabs API.
func (c *ElevenLabsClient) TTS(ctx context.Context, model, voice, text string, w io.Writer) error {
	req := &elevenlabs.TTSRequest{
		VoiceID:       voice,
		Text:          text,
		VoiceSettings: elevenlabs.DefaultVoiceSettings(),
		OutputFormat:  "mp3_44100_128",
	}
	if model != "" {
		req.ModelID = model
	}
	return c.sdk.TextToSpeech().GenerateToWriter(ctx, req, w)
}
