package providers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileProvider_Name(t *testing.T) {
	p := &FileProvider{}
	if p.Name() != "file" {
		t.Errorf("expected name=file, got %s", p.Name())
	}
}

func TestFileProvider_Validate_Read(t *testing.T) {
	p := &FileProvider{}
	args := map[string]interface{}{
		"operation": "read",
		"path":      "/tmp/test.txt",
	}
	if err := p.Validate(args); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}
}

func TestFileProvider_Validate_MissingOperation(t *testing.T) {
	p := &FileProvider{}
	args := map[string]interface{}{"path": "/tmp/test.txt"}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'operation'") {
		t.Errorf("expected missing operation error, got: %v", err)
	}
}

func TestFileProvider_Validate_InvalidOperation(t *testing.T) {
	p := &FileProvider{}
	args := map[string]interface{}{"operation": "invalid", "path": "/tmp/test.txt"}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Errorf("expected unknown operation error, got: %v", err)
	}
}

func TestFileProvider_Validate_WriteMissingContent(t *testing.T) {
	p := &FileProvider{}
	args := map[string]interface{}{"operation": "write", "path": "/tmp/test.txt"}
	err := p.Validate(args)
	if err == nil || !strings.Contains(err.Error(), "missing required parameter 'content'") {
		t.Errorf("expected missing content error, got: %v", err)
	}
}

func TestFileProvider_Run_ReadWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	p := &FileProvider{}

	// Write
	result, err := p.Run(context.Background(), map[string]interface{}{
		"operation": "write",
		"path":      path,
		"content":   "hello world",
	})
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if !strings.Contains(result.Output, "written") {
		t.Errorf("expected write confirmation, got: %s", result.Output)
	}

	// Read
	result, err = p.Run(context.Background(), map[string]interface{}{
		"operation": "read",
		"path":      path,
	})
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if result.Output != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", result.Output)
	}
}

func TestFileProvider_Run_Append(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("line1\n"), 0644)

	p := &FileProvider{}
	_, err := p.Run(context.Background(), map[string]interface{}{
		"operation": "append",
		"path":      path,
		"content":   "line2\n",
	})
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "line1\nline2\n" {
		t.Errorf("expected 'line1\\nline2\\n', got '%s'", string(data))
	}
}

func TestFileProvider_Run_CopyMove(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	cp := filepath.Join(dir, "copy.txt")
	mv := filepath.Join(dir, "moved.txt")
	os.WriteFile(src, []byte("data"), 0644)

	p := &FileProvider{}

	// Copy
	_, err := p.Run(context.Background(), map[string]interface{}{
		"operation":   "copy",
		"path":        src,
		"destination": cp,
	})
	if err != nil {
		t.Fatalf("copy failed: %v", err)
	}
	data, _ := os.ReadFile(cp)
	if string(data) != "data" {
		t.Errorf("copy content mismatch: %s", string(data))
	}

	// Move
	_, err = p.Run(context.Background(), map[string]interface{}{
		"operation":   "move",
		"path":        cp,
		"destination": mv,
	})
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if _, err := os.Stat(cp); !os.IsNotExist(err) {
		t.Error("expected source to be removed after move")
	}
	data, _ = os.ReadFile(mv)
	if string(data) != "data" {
		t.Errorf("move content mismatch: %s", string(data))
	}
}

func TestFileProvider_Run_Delete(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("data"), 0644)

	p := &FileProvider{}
	_, err := p.Run(context.Background(), map[string]interface{}{
		"operation": "delete",
		"path":      path,
	})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be deleted")
	}
}

func TestFileProvider_Run_JSONExtract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	os.WriteFile(path, []byte(`{"user":{"name":"alice","age":30}}`), 0644)

	p := &FileProvider{}
	result, err := p.Run(context.Background(), map[string]interface{}{
		"operation":  "json-extract",
		"path":       path,
		"expression": "user.name",
	})
	if err != nil {
		t.Fatalf("json-extract failed: %v", err)
	}
	if result.Output != `"alice"` {
		t.Errorf("expected '\"alice\"', got '%s'", result.Output)
	}
}

func TestFileProvider_Run_CSVToJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.csv")
	os.WriteFile(path, []byte("name,age\nalice,30\nbob,25\n"), 0644)

	p := &FileProvider{}
	result, err := p.Run(context.Background(), map[string]interface{}{
		"operation": "csv-to-json",
		"path":      path,
	})
	if err != nil {
		t.Fatalf("csv-to-json failed: %v", err)
	}

	var rows []map[string]string
	if err := json.Unmarshal([]byte(result.Output), &rows); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	if rows[0]["name"] != "alice" {
		t.Errorf("expected first row name=alice, got %s", rows[0]["name"])
	}
}
