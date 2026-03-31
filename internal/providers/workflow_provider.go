package providers

import (
	"context"
	"fmt"
)

// WorkflowExecutor is set by the models package to avoid import cycles.
var WorkflowExecutor func(ctx context.Context, name string) (map[string]*TaskResult, error)

type WorkflowProvider struct{}

func (p *WorkflowProvider) Name() string { return "workflow" }

func (p *WorkflowProvider) Validate(args map[string]interface{}) error {
	if args["workflow"] == nil || args["workflow"] == "" {
		return fmt.Errorf("workflow provider validation failed: missing required parameter 'workflow'")
	}

	for key, value := range args {
		switch key {
		case "workflow":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("workflow provider validation failed: workflow must be a string, got %T", value)
			}
		case "async":
			if _, ok := value.(bool); !ok {
				return fmt.Errorf("workflow provider validation failed: async must be a boolean, got %T", value)
			}
		default:
			return fmt.Errorf("workflow provider validation failed: unknown parameter '%s'", key)
		}
	}
	return nil
}

func (p *WorkflowProvider) Run(ctx context.Context, args map[string]interface{}) (*TaskResult, error) {
	workflowName := args["workflow"].(string)

	if WorkflowExecutor == nil {
		return nil, fmt.Errorf("workflow provider: executor not configured")
	}

	async := false
	if a, ok := args["async"].(bool); ok {
		async = a
	}

	if async {
		go WorkflowExecutor(context.Background(), workflowName) //nolint:errcheck
		return &TaskResult{
			Output:   fmt.Sprintf("workflow '%s' triggered asynchronously", workflowName),
			Metadata: map[string]string{"workflow": workflowName, "async": "true"},
		}, nil
	}

	results, err := WorkflowExecutor(ctx, workflowName)
	if err != nil {
		return nil, fmt.Errorf("workflow provider: child workflow '%s' failed: %w", workflowName, err)
	}

	// Combine child workflow outputs
	var output string
	metadata := map[string]string{"workflow": workflowName, "async": "false"}
	for taskName, result := range results {
		if result != nil && result.Output != "" {
			output += fmt.Sprintf("[%s] %s\n", taskName, result.Output)
		}
	}

	return &TaskResult{Output: output, Metadata: metadata}, nil
}

func init() {
	Register(&WorkflowProvider{})
}
