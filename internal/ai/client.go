package ai

// NOTE: This package intentionally avoids importing the OpenAI SDK in tests
// due to sandbox network restrictions. The structure is ready to wire the
// official SDK in a follow-up step without changing call sites.

import (
    "errors"
)

// Client is a thin wrapper intended to hold the official OpenAI SDK client
// instance plus minimal options needed across the app.
type Client struct {
    apiKey  string
    baseURL string
}

// New constructs a new AI client. The apiKey is required.
// baseURL is optional (use empty string for default api endpoint).
func New(apiKey, baseURL string) (*Client, error) {
    if apiKey == "" {
        return nil, errors.New("OPENAI_API_KEY is required")
    }
    return &Client{apiKey: apiKey, baseURL: baseURL}, nil
}

func (c *Client) APIKey() string  { return c.apiKey }
func (c *Client) BaseURL() string { return c.baseURL }

