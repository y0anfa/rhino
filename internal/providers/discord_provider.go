package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type DiscordProvider struct{}

func (p *DiscordProvider) Name() string { return "discord" }

func (p *DiscordProvider) Validate(args map[string]interface{}) error {
	if args["webhook-url"] == nil || args["webhook-url"] == "" {
		return fmt.Errorf("discord provider validation failed: missing required parameter 'webhook-url'")
	}
	if args["content"] == nil || args["content"] == "" {
		return fmt.Errorf("discord provider validation failed: missing required parameter 'content'")
	}

	for key, value := range args {
		switch key {
		case "webhook-url":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("discord provider validation failed: webhook-url must be a string, got %T", value)
			}
			if _, err := url.ParseRequestURI(s); err != nil {
				return fmt.Errorf("discord provider validation failed: invalid webhook-url: %w", err)
			}
		case "content":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("discord provider validation failed: content must be a string, got %T", value)
			}
		case "username":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("discord provider validation failed: username must be a string, got %T", value)
			}
		default:
			return fmt.Errorf("discord provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *DiscordProvider) Run(ctx context.Context, args map[string]interface{}) (*TaskResult, error) {
	webhookURL := args["webhook-url"].(string)
	content := args["content"].(string)

	payload := map[string]interface{}{
		"content": content,
	}
	if u, ok := args["username"].(string); ok && u != "" {
		payload["username"] = u
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("discord provider failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("discord provider failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("discord provider request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("discord provider request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return &TaskResult{
		Output: string(respBody),
		Metadata: map[string]string{
			"status_code": fmt.Sprintf("%d", resp.StatusCode),
		},
	}, nil
}

func init() {
	Register(&DiscordProvider{})
}
