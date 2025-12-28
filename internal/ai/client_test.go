package ai

import "testing"

func TestNewClientRequiresKey(t *testing.T) {
	if _, err := New("", ""); err == nil {
		t.Fatalf("expected error when api key missing")
	}
}

func TestNewClientStoresFields(t *testing.T) {
	c, err := New("sk-test", "https://example.com/v1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if c.APIKey() != "sk-test" {
		t.Fatalf("apikey mismatch")
	}
	if c.BaseURL() != "https://example.com/v1" {
		t.Fatalf("baseURL mismatch")
	}
}
