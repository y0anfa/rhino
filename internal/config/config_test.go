package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	return path
}

func TestNewConfig(t *testing.T) {
	c := NewConfig("workflows", 9090)
	if c.WorkflowsDir != "workflows" {
		t.Errorf("expected WorkflowsDir=workflows, got %s", c.WorkflowsDir)
	}
	if c.Port != 9090 {
		t.Errorf("expected Port=9090, got %d", c.Port)
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	path := setupTestConfig(t, "workflows-dir: wf\nport: 8080\n")
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.WorkflowsDir != "wf" {
		t.Errorf("expected WorkflowsDir=wf, got %s", c.WorkflowsDir)
	}
	if c.Port != 8080 {
		t.Errorf("expected Port=8080, got %d", c.Port)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	path := setupTestConfig(t, "{{invalid yaml")
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "failed to decode config") {
		t.Errorf("expected decode error, got: %v", err)
	}
}

func TestLoadConfig_ValidationFails(t *testing.T) {
	path := setupTestConfig(t, "workflows-dir: \"\"\nport: 0\n")
	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	path := setupTestConfig(t, "workflows-dir: wf\nport: 8080\n")

	t.Setenv(EnvWorkflowsDir, "override-dir")
	t.Setenv(EnvPort, "9999")

	c, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.WorkflowsDir != "override-dir" {
		t.Errorf("expected WorkflowsDir=override-dir, got %s", c.WorkflowsDir)
	}
	if c.Port != 9999 {
		t.Errorf("expected Port=9999, got %d", c.Port)
	}
}

func TestLoadConfig_InvalidPortEnv(t *testing.T) {
	path := setupTestConfig(t, "workflows-dir: wf\nport: 8080\n")

	t.Setenv(EnvPort, "not-a-number")

	_, err := LoadConfig(path)
	if err == nil || !strings.Contains(err.Error(), "invalid port") {
		t.Errorf("expected invalid port error, got: %v", err)
	}
}

func TestValidate_Valid(t *testing.T) {
	c := NewConfig("wf", 8080)
	if err := c.Validate(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestValidate_EmptyWorkflowsDir(t *testing.T) {
	c := NewConfig("", 8080)
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "workflows directory not set") {
		t.Errorf("expected workflows dir error, got: %v", err)
	}
}

func TestValidate_ZeroPort(t *testing.T) {
	c := NewConfig("wf", 0)
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "port not set") {
		t.Errorf("expected port error, got: %v", err)
	}
}

func TestValidate_PortOutOfRange(t *testing.T) {
	c := NewConfig("wf", 70000)
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "port must be between") {
		t.Errorf("expected port range error, got: %v", err)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	c := NewConfig("my-workflows", 3000)
	if err := c.Save(path); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}
	if loaded.WorkflowsDir != "my-workflows" {
		t.Errorf("expected WorkflowsDir=my-workflows, got %s", loaded.WorkflowsDir)
	}
	if loaded.Port != 3000 {
		t.Errorf("expected Port=3000, got %d", loaded.Port)
	}
}

func TestGetConfigPath_Default(t *testing.T) {
	t.Setenv(EnvConfigPath, "")
	if GetConfigPath() != DefaultConfigPath {
		t.Errorf("expected default path, got %s", GetConfigPath())
	}
}

func TestGetConfigPath_EnvOverride(t *testing.T) {
	t.Setenv(EnvConfigPath, "/custom/path.yaml")
	if GetConfigPath() != "/custom/path.yaml" {
		t.Errorf("expected /custom/path.yaml, got %s", GetConfigPath())
	}
}

func TestGetStringAndGetInt(t *testing.T) {
	Reset()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	c := NewConfig("test-wf", 4444)
	if err := c.Save(path); err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	t.Setenv(EnvConfigPath, path)

	val := GetString("workflows-dir")
	if val != "test-wf" {
		t.Errorf("expected test-wf, got %s", val)
	}

	port := GetInt("port")
	if port != 4444 {
		t.Errorf("expected 4444, got %d", port)
	}

	// Unknown keys
	if GetString("unknown") != "" {
		t.Error("expected empty for unknown key")
	}
	if GetInt("unknown") != 0 {
		t.Error("expected 0 for unknown key")
	}

	Reset()
}

func TestConfigFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv(EnvConfigPath, path)

	if ConfigFileExists() {
		t.Error("expected false before creating file")
	}

	os.WriteFile(path, []byte("workflows-dir: wf\nport: 8080\n"), 0644)
	if !ConfigFileExists() {
		t.Error("expected true after creating file")
	}
}
