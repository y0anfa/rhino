package providers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPProvider_Name(t *testing.T) {
	p := &HTTPProvider{}
	if p.Name() != "http" {
		t.Errorf("expected name=http, got %s", p.Name())
	}
}

func TestHTTPProvider_Validate_Valid(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    "https://example.com",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestHTTPProvider_Validate_WithAllParams(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method":  "POST",
		"url":     "https://example.com/api",
		"body":    `{"key":"value"}`,
		"headers": map[string]interface{}{"Content-Type": "application/json"},
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestHTTPProvider_Validate_MissingMethod(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"url": "https://example.com",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'method'") {
		t.Errorf("expected missing method error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_MissingURL(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'url'") {
		t.Errorf("expected missing url error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_InvalidMethod(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "INVALID",
		"url":    "https://example.com",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "invalid HTTP method") {
		t.Errorf("expected invalid method error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_InvalidURL(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    "not-a-url",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "invalid url") {
		t.Errorf("expected invalid url error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_BodyNotString(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "POST",
		"url":    "https://example.com",
		"body":   123,
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "body must be a string") {
		t.Errorf("expected body type error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_HeadersNotMap(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method":  "GET",
		"url":     "https://example.com",
		"headers": "not-a-map",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "headers must be a map") {
		t.Errorf("expected headers type error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_HeaderValueNotString(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method":  "GET",
		"url":     "https://example.com",
		"headers": map[string]interface{}{"Key": 123},
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "header 'Key' must be a string") {
		t.Errorf("expected header value type error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_UnknownParam(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    "https://example.com",
		"extra":  "bad",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Errorf("expected unknown param error, got: %v", err)
	}
}

func TestHTTPProvider_Run_GET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    server.URL,
	}
	result, err := p.Run(args)
	if err != nil {
		t.Errorf("expected successful run, got error: %v", err)
	}
	if result == nil || result.Output != "ok" {
		t.Errorf("expected output='ok', got: %v", result)
	}
	if result.Metadata["status_code"] != "200" {
		t.Errorf("expected status_code=200, got: %s", result.Metadata["status_code"])
	}
}

func TestHTTPProvider_Run_POST_WithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type=application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method":  "POST",
		"url":     server.URL,
		"body":    `{"key":"value"}`,
		"headers": map[string]interface{}{"Content-Type": "application/json"},
	}
	result, err := p.Run(args)
	if err != nil {
		t.Errorf("expected successful run, got error: %v", err)
	}
	if result == nil || result.Output != `{"id":1}` {
		t.Errorf("expected response body in output, got: %v", result)
	}
}

func TestHTTPProvider_Run_NoBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    server.URL,
	}
	_, err := p.Run(args)
	if err != nil {
		t.Errorf("expected successful run without body, got error: %v", err)
	}
}

func TestHTTPProvider_Run_4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    server.URL,
	}
	_, err := p.Run(args)
	if err == nil || !strings.Contains(err.Error(), "status 404") {
		t.Errorf("expected 404 error, got: %v", err)
	}
}

func TestHTTPProvider_Run_5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    server.URL,
	}
	_, err := p.Run(args)
	if err == nil || !strings.Contains(err.Error(), "status 500") {
		t.Errorf("expected 500 error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_MethodNotString(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": 123,
		"url":    "https://example.com",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "method must be a string") {
		t.Errorf("expected method type error, got: %v", err)
	}
}

func TestHTTPProvider_Validate_URLNotString(t *testing.T) {
	p := &HTTPProvider{}
	args := map[string]interface{}{
		"method": "GET",
		"url":    123,
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "url must be a string") {
		t.Errorf("expected url type error, got: %v", err)
	}
}
