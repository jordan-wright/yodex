package ai

import (
    "context"
    "errors"
    "io"

    openai "github.com/openai/openai-go/v3"
    "github.com/openai/openai-go/v3/option"
    "github.com/openai/openai-go/v3/packages/param"
    "github.com/openai/openai-go/v3/responses"
)

// Client wraps the official OpenAI SDK client and exposes minimal helpers used by the app.
type Client struct {
    apiKey  string
    baseURL string
    sdk     openai.Client
}

// New constructs a new AI client. The apiKey is required.
// baseURL is optional (empty string uses the default API endpoint).
func New(apiKey, baseURL string) (*Client, error) {
    if apiKey == "" {
        return nil, errors.New("OPENAI_API_KEY is required")
    }
    opts := []option.RequestOption{option.WithAPIKey(apiKey)}
    if baseURL != "" {
        opts = append(opts, option.WithBaseURL(baseURL))
    }
    sdk := openai.NewClient(opts...)
    return &Client{apiKey: apiKey, baseURL: baseURL, sdk: sdk}, nil
}

func (c *Client) APIKey() string  { return c.apiKey }
func (c *Client) BaseURL() string { return c.baseURL }

// GenerateText calls the Responses API and returns concatenated output text.
// The system prompt is supplied via the "instructions" field.
func (c *Client) GenerateText(ctx context.Context, model, system, prompt string) (string, error) {
    req := responses.ResponseNewParams{
        Model:        model,
        Instructions: param.NewOpt(system),
        Input:        responses.ResponseNewParamsInputUnion{OfString: param.NewOpt(prompt)},
    }
    res, err := c.sdk.Responses.New(ctx, req)
    if err != nil {
        return "", err
    }
    return res.OutputText(), nil
}

// TTS writes MP3 audio to the provided writer using the Audio Speech API.
// model should be a TTS-capable model (e.g., gpt-4o-mini-tts) and voice is a supported voice name.
func (c *Client) TTS(ctx context.Context, model, voice, text string, w io.Writer) error {
    req := openai.AudioSpeechNewParams{
        Model:          openai.SpeechModel(model),
        Voice:          openai.AudioSpeechNewParamsVoice(voice),
        Input:          text,
        ResponseFormat: openai.AudioSpeechNewParamsResponseFormatMP3,
    }
    resp, err := c.sdk.Audio.Speech.New(ctx, req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    _, err = io.Copy(w, resp.Body)
    return err
}
