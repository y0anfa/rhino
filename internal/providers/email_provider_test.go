package providers

import (
	"strings"
	"testing"
)

func TestEmailProvider_Name(t *testing.T) {
	p := &EmailProvider{}
	if p.Name() != "email" {
		t.Errorf("expected name=email, got %s", p.Name())
	}
}

func TestEmailProvider_Validate_Valid(t *testing.T) {
	p := &EmailProvider{}
	args := map[string]interface{}{
		"smtp-host": "smtp.example.com",
		"smtp-port": "587",
		"from":      "test@example.com",
		"to":        "user@example.com",
		"subject":   "Test",
		"body":      "Hello",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestEmailProvider_Validate_ValidWithList(t *testing.T) {
	p := &EmailProvider{}
	args := map[string]interface{}{
		"smtp-host": "smtp.example.com",
		"smtp-port": "587",
		"from":      "test@example.com",
		"to":        []interface{}{"a@b.com", "c@d.com"},
		"subject":   "Test",
		"body":      "Hello",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestEmailProvider_Validate_MissingRequired(t *testing.T) {
	p := &EmailProvider{}
	args := map[string]interface{}{
		"smtp-host": "smtp.example.com",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter") {
		t.Errorf("expected missing param error, got: %v", err)
	}
}

func TestEmailProvider_Validate_UnknownParam(t *testing.T) {
	p := &EmailProvider{}
	args := map[string]interface{}{
		"smtp-host": "smtp.example.com",
		"smtp-port": "587",
		"from":      "test@example.com",
		"to":        "user@example.com",
		"subject":   "Test",
		"body":      "Hello",
		"extra":     "bad",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Errorf("expected unknown param error, got: %v", err)
	}
}
