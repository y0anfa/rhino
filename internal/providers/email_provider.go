package providers

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
)

type EmailProvider struct{}

func (p *EmailProvider) Name() string { return "email" }

func (p *EmailProvider) Validate(args map[string]interface{}) error {
	required := []string{"smtp-host", "smtp-port", "from", "to", "subject", "body"}
	for _, param := range required {
		if args[param] == nil || args[param] == "" {
			return fmt.Errorf("email provider validation failed: missing required parameter '%s'", param)
		}
	}

	for key, value := range args {
		switch key {
		case "smtp-host", "from", "subject", "body", "username", "password":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("email provider validation failed: %s must be a string, got %T", key, value)
			}
		case "smtp-port":
			if _, ok := value.(string); !ok {
				// YAML may parse port as int
				if _, ok := value.(int); !ok {
					return fmt.Errorf("email provider validation failed: smtp-port must be a string or int, got %T", value)
				}
			}
		case "to":
			switch v := value.(type) {
			case string:
				// single recipient
			case []interface{}:
				for _, item := range v {
					if _, ok := item.(string); !ok {
						return fmt.Errorf("email provider validation failed: to list items must be strings, got %T", item)
					}
				}
			default:
				return fmt.Errorf("email provider validation failed: to must be a string or list, got %T", value)
			}
		case "html":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("email provider validation failed: html must be a boolean, got %T", value)
			}
		default:
			return fmt.Errorf("email provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *EmailProvider) Run(_ context.Context, args map[string]interface{}) (*TaskResult, error) {
	host := args["smtp-host"].(string)
	port := fmt.Sprintf("%v", args["smtp-port"])
	from := args["from"].(string)
	subject := args["subject"].(string)
	body := args["body"].(string)

	var recipients []string
	switch v := args["to"].(type) {
	case string:
		recipients = []string{v}
	case []interface{}:
		for _, item := range v {
			recipients = append(recipients, item.(string))
		}
	}

	isHTML := false
	if h, ok := args["html"].(bool); ok {
		isHTML = h
	}

	contentType := "text/plain"
	if isHTML {
		contentType = "text/html"
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: %s; charset=UTF-8\r\n\r\n%s",
		from, strings.Join(recipients, ","), subject, contentType, body)

	addr := net.JoinHostPort(host, port)

	var auth smtp.Auth
	if u, ok := args["username"].(string); ok && u != "" {
		pw, _ := args["password"].(string)
		auth = smtp.PlainAuth("", u, pw, host)
	}

	// Try TLS first
	tlsConfig := &tls.Config{ServerName: host}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err == nil {
		client, err := smtp.NewClient(conn, host)
		if err == nil {
			defer client.Close()
			if auth != nil {
				if err := client.Auth(auth); err != nil {
					return nil, fmt.Errorf("email provider auth failed: %w", err)
				}
			}
			if err := client.Mail(from); err != nil {
				return nil, fmt.Errorf("email provider MAIL FROM failed: %w", err)
			}
			for _, r := range recipients {
				if err := client.Rcpt(r); err != nil {
					return nil, fmt.Errorf("email provider RCPT TO failed: %w", err)
				}
			}
			w, err := client.Data()
			if err != nil {
				return nil, fmt.Errorf("email provider DATA failed: %w", err)
			}
			if _, err := w.Write([]byte(msg)); err != nil {
				return nil, fmt.Errorf("email provider write failed: %w", err)
			}
			if err := w.Close(); err != nil {
				return nil, fmt.Errorf("email provider close failed: %w", err)
			}
			return &TaskResult{
				Output:   "email sent successfully",
				Metadata: map[string]string{"recipients": strings.Join(recipients, ",")},
			}, nil
		}
	}

	// Fallback to plain SMTP
	if err := smtp.SendMail(addr, auth, from, recipients, []byte(msg)); err != nil {
		return nil, fmt.Errorf("email provider failed to send: %w", err)
	}

	return &TaskResult{
		Output:   "email sent successfully",
		Metadata: map[string]string{"recipients": strings.Join(recipients, ",")},
	}, nil
}

func init() {
	Register(&EmailProvider{})
}
