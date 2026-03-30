package template

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/y0anfa/rhino/internal/providers"
)

var templateRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)

type Context struct {
	WorkflowName string
	WorkflowDesc string
	TriggerName  string
	TriggerType  string
	TaskResults  map[string]*providers.TaskResult
}

func NewContext(workflowName, workflowDesc, triggerName, triggerType string) *Context {
	return &Context{
		WorkflowName: workflowName,
		WorkflowDesc: workflowDesc,
		TriggerName:  triggerName,
		TriggerType:  triggerType,
		TaskResults:  make(map[string]*providers.TaskResult),
	}
}

func (c *Context) SetTaskResult(taskName string, result *providers.TaskResult) {
	c.TaskResults[taskName] = result
}

func (c *Context) Resolve(input string) string {
	return templateRegex.ReplaceAllStringFunc(input, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		if val, ok := c.resolve(expr); ok {
			return val
		}
		return match
	})
}

func (c *Context) resolve(expr string) (string, bool) {
	switch {
	case strings.HasPrefix(expr, "env."):
		key := expr[4:]
		return os.Getenv(key), true

	case strings.HasPrefix(expr, "task."):
		return c.resolveTaskExpr(expr[5:])

	case expr == "workflow.name":
		return c.WorkflowName, true
	case expr == "workflow.description":
		return c.WorkflowDesc, true

	case expr == "trigger.name":
		return c.TriggerName, true
	case expr == "trigger.type":
		return c.TriggerType, true

	case expr == "timestamp":
		return time.Now().UTC().Format(time.RFC3339), true
	case expr == "date":
		return time.Now().UTC().Format("2006-01-02"), true

	case strings.HasPrefix(expr, "secret."):
		// Secrets are resolved elsewhere; leave placeholder
		return "", false
	}

	return "", false
}

func (c *Context) resolveTaskExpr(expr string) (string, bool) {
	// expr is like "TASKNAME.output" or "TASKNAME.metadata.KEY"
	parts := strings.SplitN(expr, ".", 2)
	if len(parts) < 2 {
		return "", false
	}
	taskName := parts[0]
	field := parts[1]

	result, ok := c.TaskResults[taskName]
	if !ok || result == nil {
		return "", false
	}

	switch {
	case field == "output":
		return strings.TrimSpace(result.Output), true
	case strings.HasPrefix(field, "metadata."):
		key := field[9:]
		if val, ok := result.Metadata[key]; ok {
			return val, true
		}
		return "", true
	}

	return "", false
}

func ResolveParams(params map[string]interface{}, ctx *Context) map[string]interface{} {
	resolved := make(map[string]interface{}, len(params))
	for k, v := range params {
		resolved[k] = resolveValue(v, ctx)
	}
	return resolved
}

func resolveValue(v interface{}, ctx *Context) interface{} {
	switch val := v.(type) {
	case string:
		return ctx.Resolve(val)
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = resolveValue(item, ctx)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		for k, item := range val {
			result[k] = resolveValue(item, ctx)
		}
		return result
	default:
		return v
	}
}

func ValidateTemplateRefs(params map[string]interface{}, priorTasks []string) []string {
	var warnings []string
	for _, v := range params {
		warnings = append(warnings, validateValueRefs(v, priorTasks)...)
	}
	return warnings
}

func validateValueRefs(v interface{}, priorTasks []string) []string {
	var warnings []string
	s, ok := v.(string)
	if !ok {
		return nil
	}

	matches := templateRegex.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		expr := strings.TrimSpace(match[1])
		if strings.HasPrefix(expr, "task.") {
			parts := strings.SplitN(expr[5:], ".", 2)
			if len(parts) >= 1 {
				taskName := parts[0]
				found := false
				for _, pt := range priorTasks {
					if pt == taskName {
						found = true
						break
					}
				}
				if !found {
					warnings = append(warnings, fmt.Sprintf("template ref '{{%s}}' references task '%s' not in a prior order group", expr, taskName))
				}
			}
		}
	}
	return warnings
}
