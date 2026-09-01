package providers

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestShellProvider_Name(t *testing.T) {
	p := &ShellProvider{}
	if p.Name() != "shell" {
		t.Errorf("expected name=shell, got %s", p.Name())
	}
}

func TestShellProvider_Validate_Valid(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello"},
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestShellProvider_Validate_MissingCommand(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"args": []interface{}{"hello"},
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'command'") {
		t.Errorf("expected missing command error, got: %v", err)
	}
}

func TestShellProvider_Validate_ArgsOptional(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected args to be optional, got: %v", err)
	}
}

func TestShellProvider_Validate_CommandNotString(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": 123,
		"args":    []interface{}{"hello"},
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "command must be a string") {
		t.Errorf("expected type error, got: %v", err)
	}
}

func TestShellProvider_Validate_ArgsNotList(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
		"args":    "not-a-list",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "args must be a list") {
		t.Errorf("expected type error, got: %v", err)
	}
}

func TestShellProvider_Validate_ArgsNotStrings(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{123},
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "args must be strings") {
		t.Errorf("expected type error, got: %v", err)
	}
}

func TestShellProvider_Validate_UnknownParam(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello"},
		"extra":   "bad",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Errorf("expected unknown param error, got: %v", err)
	}
}

func TestShellProvider_Validate_EmptyCommand(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "",
		"args":    []interface{}{},
	}
	err := p.Validate(args)
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestShellProvider_Run(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"hello"},
	}
	result, err := p.Run(context.Background(), args)
	if err != nil {
		t.Errorf("expected successful run, got error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected output to contain 'hello', got: %s", result.Output)
	}
}

func TestShellProvider_Run_Failure(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "false",
		"args":    []interface{}{},
	}
	_, err := p.Run(context.Background(), args)
	if err == nil {
		t.Error("expected error from failing command")
	}
}

func TestShellProvider_Run_WithoutArgs(t *testing.T) {
	p := &ShellProvider{}
	res, err := p.Run(context.Background(), map[string]interface{}{"command": "true"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Metadata["exit_code"] != "0" {
		t.Errorf("expected exit_code metadata '0', got %q", res.Metadata["exit_code"])
	}
}

func TestShellProvider_Run_FailureIncludesStderr(t *testing.T) {
	p := &ShellProvider{}
	_, err := p.Run(context.Background(), map[string]interface{}{
		"command": "sh",
		"args":    []interface{}{"-c", "echo boom >&2; exit 3"},
	})
	if err == nil {
		t.Fatal("expected an error for a non-zero exit")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected stderr in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "exit status 3") {
		t.Errorf("expected exit status in error, got: %v", err)
	}
}

func TestShellProvider_Run_TimeoutReportsDeadline(t *testing.T) {
	p := &ShellProvider{}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := p.Run(ctx, map[string]interface{}{
		"command": "sleep",
		"args":    []interface{}{"5"},
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got: %v", err)
	}
}
