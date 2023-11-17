package providers

import (
	"fmt"
	"net/http"
	"strings"
)

type HTTPProvider struct{}

func (p *HTTPProvider) Name() string {
	return "http"
}

func (p *HTTPProvider) Validate(args map[string]interface{}) error {
	requiredParams := []string{"method", "url", "body"}
	for _, param := range requiredParams {
		if args[param] == "" {
			return fmt.Errorf("missing %s parameter", param)
		}
	}

	for key, value := range args {
		switch key {
		case "method":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid method parameter")
			}
		case "url":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid url parameter")
			}
		case "body":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("invalid body parameter")
			}
		case "headers":
			if _, ok := value.(map[string]string); !ok {
				return fmt.Errorf("invalid headers parameter")
			}
		default:
			return fmt.Errorf("unknown parameter: %s", key)
		}
	}
	return nil
}

func (p *HTTPProvider) Run(args map[string]interface{}) error {
	err := p.Validate(args)
	if err != nil {
		return err
	}

	method := strings.ToUpper(args["method"].(string))
	url := args["url"].(string)
	body := args["body"].(string)
	
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		return err
	}
	if args["headers"] != "" {
		headers, ok := args["headers"].(map[string]string)
		if !ok {
			return fmt.Errorf("invalid headers parameter")
		}
		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	fmt.Println(resp.Status)
	return nil
}

func init() {
	Register(&HTTPProvider{})
}