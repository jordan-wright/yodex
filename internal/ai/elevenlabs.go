package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const elevenLabsDefaultBaseURL = "https://api.elevenlabs.io"
const elevenLabsDefaultOutputFormat = "mp3_44100_128"

// ElevenLabsOption configures the ElevenLabs client.
type ElevenLabsOption func(*ElevenLabsClient)

// WithElevenLabsBaseURL sets the ElevenLabs API base URL.
func WithElevenLabsBaseURL(baseURL string) ElevenLabsOption {
	return func(c *ElevenLabsClient) {
		if baseURL != "" {
			c.baseURL = baseURL
		}
	}
}

// WithElevenLabsHTTPClient sets the HTTP client used for requests.
func WithElevenLabsHTTPClient(client *http.Client) ElevenLabsOption {
	return func(c *ElevenLabsClient) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// ElevenLabsClient provides a thin wrapper for ElevenLabs API calls.
type ElevenLabsClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	tts        *ElevenLabsTextToSpeechService
}

// NewElevenLabs constructs a new ElevenLabs client. The apiKey is required.
func NewElevenLabs(apiKey string, opts ...ElevenLabsOption) (*ElevenLabsClient, error) {
	if apiKey == "" {
		return nil, errors.New("ELEVENLABS_API_KEY is required")
	}
	client := &ElevenLabsClient{
		apiKey:  apiKey,
		baseURL: elevenLabsDefaultBaseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
	for _, opt := range opts {
		opt(client)
	}
	client.tts = &ElevenLabsTextToSpeechService{client: client}
	return client, nil
}

// TextToSpeech returns the text-to-speech service.
func (c *ElevenLabsClient) TextToSpeech() *ElevenLabsTextToSpeechService {
	return c.tts
}

// ElevenLabsVoiceSettings configures TTS voice settings.
type ElevenLabsVoiceSettings struct {
	Stability       float64 `json:"stability,omitempty"`
	SimilarityBoost float64 `json:"similarity_boost,omitempty"`
	Style           float64 `json:"style,omitempty"`
	UseSpeakerBoost bool    `json:"use_speaker_boost,omitempty"`
}

// DefaultElevenLabsVoiceSettings returns recommended defaults.
func DefaultElevenLabsVoiceSettings() *ElevenLabsVoiceSettings {
	return &ElevenLabsVoiceSettings{
		Stability:       0.0,
		SimilarityBoost: 0.75,
		Style:           0.0,
		UseSpeakerBoost: true,
	}
}

// ElevenLabsTTSRequest is a request to generate speech.
type ElevenLabsTTSRequest struct {
	VoiceID       string
	Text          string
	ModelID       string
	VoiceSettings *ElevenLabsVoiceSettings
	OutputFormat  string
}

// ElevenLabsTextToSpeechService handles text-to-speech requests.
type ElevenLabsTextToSpeechService struct {
	client *ElevenLabsClient
}

// Convert generates speech audio and returns a reader for the audio stream.
func (s *ElevenLabsTextToSpeechService) Convert(ctx context.Context, req *ElevenLabsTTSRequest) (io.ReadCloser, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}
	if strings.TrimSpace(req.VoiceID) == "" {
		return nil, errors.New("voice_id is required")
	}
	if strings.TrimSpace(req.Text) == "" {
		return nil, errors.New("text is required")
	}

	outputFormat := req.OutputFormat
	if outputFormat == "" {
		outputFormat = elevenLabsDefaultOutputFormat
	}

	endpoint, err := url.Parse(strings.TrimRight(s.client.baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse elevenlabs base url: %w", err)
	}
	endpoint.Path = fmt.Sprintf("/v1/text-to-speech/%s", req.VoiceID)
	query := endpoint.Query()
	query.Set("output_format", outputFormat)
	endpoint.RawQuery = query.Encode()

	body := struct {
		Text          string                   `json:"text"`
		ModelID       string                   `json:"model_id,omitempty"`
		VoiceSettings *ElevenLabsVoiceSettings `json:"voice_settings,omitempty"`
	}{
		Text:          req.Text,
		ModelID:       req.ModelID,
		VoiceSettings: req.VoiceSettings,
	}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("encode elevenlabs request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), &buf)
	if err != nil {
		return nil, fmt.Errorf("build elevenlabs request: %w", err)
	}
	httpReq.Header.Set("xi-api-key", s.client.apiKey)
	httpReq.Header.Set("accept", "audio/mpeg")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, &ElevenLabsAPIError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       strings.TrimSpace(string(errBody)),
		}
	}
	return resp.Body, nil
}

// ConvertToWriter generates speech audio and writes it to the writer.
func (s *ElevenLabsTextToSpeechService) ConvertToWriter(ctx context.Context, req *ElevenLabsTTSRequest, w io.Writer) error {
	reader, err := s.Convert(ctx, req)
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(w, reader)
	return err
}

// ElevenLabsAPIError captures error details from ElevenLabs responses.
type ElevenLabsAPIError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *ElevenLabsAPIError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("elevenlabs api error: %s", e.Status)
	}
	return fmt.Sprintf("elevenlabs api error: %s: %s", e.Status, e.Body)
}

// TTS writes MP3 audio to the provided writer using the ElevenLabs API.
func (c *ElevenLabsClient) TTS(ctx context.Context, model, voice, text string, w io.Writer) error {
	req := &ElevenLabsTTSRequest{
		VoiceID:       voice,
		Text:          text,
		ModelID:       model,
		VoiceSettings: DefaultElevenLabsVoiceSettings(),
		OutputFormat:  elevenLabsDefaultOutputFormat,
	}
	return c.TextToSpeech().ConvertToWriter(ctx, req, w)
}
