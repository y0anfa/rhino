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

func (p *HTTPProvider) Run(args []string) error {
	// args[0] should be the method (GET, POST, etc.)
	// args[1] should be the URL
	// args[2:] could be optional parameters, like headers or body content
	// Use the net/http package to make the request
	if len(args) < 2 {
		return fmt.Errorf("no URL specified")
	}
	method := args[0]
	url := args[1]
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	// Add any optional parameters.
	for _, arg := range args[2:] {
		// Split the argument by the first colon.
		// The first part is the header name, the second part is the header value.
		parts := strings.SplitN(arg, ":", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid header: %s", arg)
		}
		// Add the header to the request.
		req.Header.Add(parts[0], parts[1])
	}
	// Send the request.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	// Print the response.
	fmt.Println(resp.Status)
	return nil
}

func init() {
	Register(&HTTPProvider{})
}