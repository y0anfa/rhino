package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscordProvider_Name(t *testing.T) {
	p := &DiscordProvider{}
	if p.Name() != "discord" {
		t.Errorf("expected name=discord, got %s", p.Name())
	}
}

func TestDiscordProvider_Validate_Valid(t *testing.T) {
	p := &DiscordProvider{}
	args := map[string]interface{}{
		"webhook-url": "https://discord.com/api/webhooks/xxx",
		"content":     "hello",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestDiscordProvider_Validate_MissingContent(t *testing.T) {
	p := &DiscordProvider{}
	args := map[string]interface{}{
		"webhook-url": "https://discord.com/api/webhooks/xxx",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'content'") {
		t.Errorf("expected missing content error, got: %v", err)
	}
}

func TestDiscordProvider_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	p := &DiscordProvider{}
	args := map[string]interface{}{
		"webhook-url": server.URL,
		"content":     "test message",
		"username":    "rhino-bot",
	}
	result, err := p.Run(context.Background(), args)
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
