package providers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type FileProvider struct{}

func (p *FileProvider) Name() string { return "file" }

func (p *FileProvider) Validate(args map[string]interface{}) error {
	if args["operation"] == nil || args["operation"] == "" {
		return fmt.Errorf("file provider validation failed: missing required parameter 'operation'")
	}

	op, ok := args["operation"].(string)
	if !ok {
		return fmt.Errorf("file provider validation failed: operation must be a string, got %T", args["operation"])
	}

	validOps := map[string]bool{
		"read": true, "write": true, "append": true,
		"copy": true, "move": true, "delete": true,
		"json-extract": true, "csv-to-json": true,
	}
	if !validOps[op] {
		return fmt.Errorf("file provider validation failed: unknown operation '%s'", op)
	}

	needsPath := map[string]bool{"read": true, "write": true, "append": true, "copy": true, "move": true, "delete": true, "json-extract": true, "csv-to-json": true}
	if needsPath[op] {
		if args["path"] == nil || args["path"] == "" {
			return fmt.Errorf("file provider validation failed: missing required parameter 'path' for operation '%s'", op)
		}
	}

	needsContent := map[string]bool{"write": true, "append": true}
	if needsContent[op] {
		if args["content"] == nil {
			return fmt.Errorf("file provider validation failed: missing required parameter 'content' for operation '%s'", op)
		}
	}

	needsDest := map[string]bool{"copy": true, "move": true}
	if needsDest[op] {
		if args["destination"] == nil || args["destination"] == "" {
			return fmt.Errorf("file provider validation failed: missing required parameter 'destination' for operation '%s'", op)
		}
	}

	if op == "json-extract" {
		if args["expression"] == nil || args["expression"] == "" {
			return fmt.Errorf("file provider validation failed: missing required parameter 'expression' for operation 'json-extract'")
		}
	}

	for key := range args {
		switch key {
		case "operation", "path", "content", "destination", "expression":
			// valid
		default:
			return fmt.Errorf("file provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *FileProvider) Run(_ context.Context, args map[string]interface{}) (*TaskResult, error) {
	op := args["operation"].(string)
	path, _ := args["path"].(string)

	switch op {
	case "read":
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("file provider read failed: %w", err)
		}
		return &TaskResult{Output: string(data)}, nil

	case "write":
		content := fmt.Sprintf("%v", args["content"])
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return nil, fmt.Errorf("file provider write failed: %w", err)
		}
		return &TaskResult{Output: "file written successfully", Metadata: map[string]string{"path": path}}, nil

	case "append":
		content := fmt.Sprintf("%v", args["content"])
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("file provider append failed: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(content); err != nil {
			return nil, fmt.Errorf("file provider append write failed: %w", err)
		}
		return &TaskResult{Output: "content appended successfully", Metadata: map[string]string{"path": path}}, nil

	case "copy":
		dest := args["destination"].(string)
		src, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("file provider copy failed to open source: %w", err)
		}
		defer src.Close()
		dst, err := os.Create(dest)
		if err != nil {
			return nil, fmt.Errorf("file provider copy failed to create destination: %w", err)
		}
		defer dst.Close()
		if _, err := io.Copy(dst, src); err != nil {
			return nil, fmt.Errorf("file provider copy failed: %w", err)
		}
		return &TaskResult{Output: "file copied successfully", Metadata: map[string]string{"source": path, "destination": dest}}, nil

	case "move":
		dest := args["destination"].(string)
		if err := os.Rename(path, dest); err != nil {
			return nil, fmt.Errorf("file provider move failed: %w", err)
		}
		return &TaskResult{Output: "file moved successfully", Metadata: map[string]string{"source": path, "destination": dest}}, nil

	case "delete":
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("file provider delete failed: %w", err)
		}
		return &TaskResult{Output: "file deleted successfully", Metadata: map[string]string{"path": path}}, nil

	case "json-extract":
		expression := args["expression"].(string)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("file provider json-extract read failed: %w", err)
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("file provider json-extract parse failed: %w", err)
		}
		// Simple dot-notation extraction
		val := extractJSONPath(obj, expression)
		output, _ := json.Marshal(val)
		return &TaskResult{Output: string(output)}, nil

	case "csv-to-json":
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("file provider csv-to-json open failed: %w", err)
		}
		defer f.Close()
		reader := csv.NewReader(f)
		records, err := reader.ReadAll()
		if err != nil {
			return nil, fmt.Errorf("file provider csv-to-json read failed: %w", err)
		}
		if len(records) < 1 {
			return &TaskResult{Output: "[]"}, nil
		}
		headers := records[0]
		var rows []map[string]string
		for _, record := range records[1:] {
			row := make(map[string]string)
			for i, header := range headers {
				if i < len(record) {
					row[header] = record[i]
				}
			}
			rows = append(rows, row)
		}
		output, _ := json.Marshal(rows)
		return &TaskResult{Output: string(output)}, nil
	}

	return nil, fmt.Errorf("file provider: unsupported operation '%s'", op)
}

func extractJSONPath(obj map[string]interface{}, path string) interface{} {
	parts := strings.Split(path, ".")
	var current interface{} = obj
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil
		}
		current = m[part]
	}
	return current
}

func init() {
	Register(&FileProvider{})
}
