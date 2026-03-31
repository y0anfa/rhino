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

type SlackProvider struct{}

func (p *SlackProvider) Name() string { return "slack" }

func (p *SlackProvider) Validate(args map[string]interface{}) error {
	if args["webhook-url"] == nil || args["webhook-url"] == "" {
		return fmt.Errorf("slack provider validation failed: missing required parameter 'webhook-url'")
	}
	if args["message"] == nil || args["message"] == "" {
		return fmt.Errorf("slack provider validation failed: missing required parameter 'message'")
	}

	for key, value := range args {
		switch key {
		case "webhook-url":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("slack provider validation failed: webhook-url must be a string, got %T", value)
			}
			if _, err := url.ParseRequestURI(s); err != nil {
				return fmt.Errorf("slack provider validation failed: invalid webhook-url: %w", err)
			}
		case "message":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("slack provider validation failed: message must be a string, got %T", value)
			}
		case "channel":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("slack provider validation failed: channel must be a string, got %T", value)
			}
		case "username":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("slack provider validation failed: username must be a string, got %T", value)
			}
		default:
			return fmt.Errorf("slack provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *SlackProvider) Run(ctx context.Context, args map[string]interface{}) (*TaskResult, error) {
	webhookURL := args["webhook-url"].(string)
	message := args["message"].(string)

	payload := map[string]interface{}{
		"text": message,
	}
	if ch, ok := args["channel"].(string); ok && ch != "" {
		payload["channel"] = ch
	}
	if u, ok := args["username"].(string); ok && u != "" {
		payload["username"] = u
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("slack provider failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("slack provider failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("slack provider request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("slack provider request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return &TaskResult{
		Output: string(respBody),
		Metadata: map[string]string{
			"status_code": fmt.Sprintf("%d", resp.StatusCode),
		},
	}, nil
}

func init() {
	Register(&SlackProvider{})
}
