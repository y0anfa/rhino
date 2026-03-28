package providers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/y0anfa/rhino/internal/logger"
	"go.uber.org/zap"
)

type HTTPProvider struct{}

func (p *HTTPProvider) Name() string {
	return "http"
}

func (p *HTTPProvider) Validate(args map[string]interface{}) error {
	requiredParams := []string{"method", "url"}
	for _, param := range requiredParams {
		if args[param] == nil || args[param] == "" {
			return fmt.Errorf("http provider validation failed: missing required parameter '%s'", param)
		}
	}

	for key, value := range args {
		switch key {
		case "method":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("http provider validation failed: method must be a string, got %T", value)
			}
			method := strings.ToUpper(value.(string))
			if method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
				return fmt.Errorf("http provider validation failed: invalid HTTP method '%s'", method)
			}
		case "url":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("http provider validation failed: url must be a string, got %T", value)
			}
			if _, err := url.ParseRequestURI(value.(string)); err != nil {
				return fmt.Errorf("http provider validation failed: invalid url '%s': %w", value.(string), err)
			}
		case "body":
			if value != nil && value != "" {
				if _, ok := value.(string); !ok {
					return fmt.Errorf("http provider validation failed: body must be a string, got %T", value)
				}
			}
		case "headers":
			if value != nil {
				if _, ok := value.(map[string]interface{}); !ok {
					return fmt.Errorf("http provider validation failed: headers must be a map, got %T", value)
				}
				for headerKey, headerValue := range value.(map[string]interface{}) {
					if _, ok := headerValue.(string); !ok {
						return fmt.Errorf("http provider validation failed: header '%s' must be a string, got %T", headerKey, headerValue)
					}
				}
			}
		default:
			return fmt.Errorf("http provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *HTTPProvider) Run(args map[string]interface{}) (*TaskResult, error) {
	method := strings.ToUpper(args["method"].(string))
	reqURL := args["url"].(string)

	var body string
	if b, ok := args["body"]; ok && b != nil {
		body, _ = b.(string)
	}

	req, err := http.NewRequest(method, reqURL, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	if args["headers"] != nil {
		for key, value := range args["headers"].(map[string]interface{}) {
			req.Header.Add(key, value.(string))
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	logger.Info("http request completed", zap.Int("status_code", resp.StatusCode), zap.String("status", resp.Status))
	return &TaskResult{
		Output: string(respBody),
		Metadata: map[string]string{
			"status_code": strconv.Itoa(resp.StatusCode),
			"status":      resp.Status,
		},
	}, nil
}

func init() {
	Register(&HTTPProvider{})
}
