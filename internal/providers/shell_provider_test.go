package providers

import (
	"strings"
	"testing"
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

func TestShellProvider_Validate_MissingArgs(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "echo",
	}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'args'") {
		t.Errorf("expected missing args error, got: %v", err)
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
	if err := p.Run(args); err != nil {
		t.Errorf("expected successful run, got error: %v", err)
	}
}

func TestShellProvider_Run_Failure(t *testing.T) {
	p := &ShellProvider{}
	args := map[string]interface{}{
		"command": "false",
		"args":    []interface{}{},
	}
	if err := p.Run(args); err == nil {
		t.Error("expected error from failing command")
	}
}
