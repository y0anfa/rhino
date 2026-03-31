package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSlackProvider_Name(t *testing.T) {
	p := &SlackProvider{}
	if p.Name() != "slack" {
		t.Errorf("expected name=slack, got %s", p.Name())
	}
}

func TestSlackProvider_Validate_Valid(t *testing.T) {
	p := &SlackProvider{}
	args := map[string]interface{}{
		"webhook-url": "https://hooks.slack.com/services/xxx",
		"message":     "hello",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestSlackProvider_Validate_MissingWebhookURL(t *testing.T) {
	p := &SlackProvider{}
	args := map[string]interface{}{
		"message": "hello",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'webhook-url'") {
		t.Errorf("expected missing webhook-url error, got: %v", err)
	}
}

func TestSlackProvider_Validate_MissingMessage(t *testing.T) {
	p := &SlackProvider{}
	args := map[string]interface{}{
		"webhook-url": "https://hooks.slack.com/services/xxx",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'message'") {
		t.Errorf("expected missing message error, got: %v", err)
	}
}

func TestSlackProvider_Validate_UnknownParam(t *testing.T) {
	p := &SlackProvider{}
	args := map[string]interface{}{
		"webhook-url": "https://hooks.slack.com/services/xxx",
		"message":     "hello",
		"extra":       "bad",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Errorf("expected unknown param error, got: %v", err)
	}
}

func TestSlackProvider_Run(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	p := &SlackProvider{}
	args := map[string]interface{}{
		"webhook-url": server.URL,
		"message":     "test message",
		"channel":     "#general",
	}
	result, err := p.Run(context.Background(), args)
	if err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
	if result.Output != "ok" {
		t.Errorf("expected output='ok', got '%s'", result.Output)
	}
}

func TestSlackProvider_Run_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("invalid_token"))
	}))
	defer server.Close()

	p := &SlackProvider{}
	args := map[string]interface{}{
		"webhook-url": server.URL,
		"message":     "test",
	}
	_, err := p.Run(context.Background(), args)
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Errorf("expected 403 error, got: %v", err)
	}
}
