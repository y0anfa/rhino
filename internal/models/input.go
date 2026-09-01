package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Input declares a value a workflow accepts at trigger time. Values arrive from
// `rhino run --input`, a webhook query string or JSON body, the dashboard API,
// or a parent workflow, and are referenced in templates as {{input.NAME}}.
type Input struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

// ErrInvalidInput marks a trigger that supplied bad inputs, so callers such as
// the webhook handler can answer with a client error instead of a server one.
var ErrInvalidInput = errors.New("invalid input")

var inputNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

var inputRefRe = regexp.MustCompile(`\{\{\s*input\.([^}\s]+)\s*\}\}`)

type inputsKey struct{}

// WithInputs attaches trigger-time input values to the context of a Run.
func WithInputs(ctx context.Context, inputs map[string]string) context.Context {
	return context.WithValue(ctx, inputsKey{}, inputs)
}

func inputsFromContext(ctx context.Context) map[string]string {
	inputs, _ := ctx.Value(inputsKey{}).(map[string]string)
	return inputs
}

// validateInputs checks the input declarations and that every {{input.X}}
// reference in the workflow points at a declared input.
func (w *Workflow) validateInputs() error {
	for name, in := range w.Inputs {
		if !inputNameRe.MatchString(name) {
			return fmt.Errorf("invalid input name '%s': use letters, digits, '_' or '-'", name)
		}
		if in.Required && in.Default != "" {
			return fmt.Errorf("input '%s' cannot be both required and have a default", name)
		}
	}

	check := func(where string, params map[string]interface{}) error {
		for _, ref := range collectInputRefs(params) {
			if _, ok := w.Inputs[ref]; !ok {
				return fmt.Errorf("%s references undeclared input '%s'", where, ref)
			}
		}
		return nil
	}
	for _, t := range w.Tasks {
		if err := check(fmt.Sprintf("task '%s'", t.Name), t.Params); err != nil {
			return err
		}
	}
	if w.Notifications != nil {
		for _, ch := range append(append([]NotificationChannel{}, w.Notifications.OnSuccess...), w.Notifications.OnFailure...) {
			if err := check(fmt.Sprintf("notification '%s'", ch.Provider), ch.Params); err != nil {
				return err
			}
		}
	}
	return nil
}

func collectInputRefs(v interface{}) []string {
	var refs []string
	switch val := v.(type) {
	case string:
		for _, m := range inputRefRe.FindAllStringSubmatch(val, -1) {
			refs = append(refs, m[1])
		}
	case []interface{}:
		for _, item := range val {
			refs = append(refs, collectInputRefs(item)...)
		}
	case map[string]interface{}:
		for _, item := range val {
			refs = append(refs, collectInputRefs(item)...)
		}
	}
	return refs
}

// resolveInputs merges provided values over declared defaults. Missing required
// inputs are an error. Undeclared values are an error when strict (CLI, API,
// parent workflows) and ignored otherwise (webhook payloads carry extra fields).
func (w *Workflow) resolveInputs(provided map[string]string, strict bool) (map[string]string, error) {
	resolved := make(map[string]string, len(w.Inputs))
	for name, in := range w.Inputs {
		resolved[name] = in.Default
	}

	var unknown []string
	for name, value := range provided {
		if _, ok := w.Inputs[name]; !ok {
			unknown = append(unknown, name)
			continue
		}
		resolved[name] = value
	}
	if strict && len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("%w: workflow '%s' does not declare input(s) %s", ErrInvalidInput, w.Name, strings.Join(unknown, ", "))
	}

	var missing []string
	for name, in := range w.Inputs {
		if in.Required {
			if _, ok := provided[name]; !ok {
				missing = append(missing, name)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("%w: workflow '%s' requires input(s) %s", ErrInvalidInput, w.Name, strings.Join(missing, ", "))
	}
	return resolved, nil
}

// InputsFromJSON flattens a JSON object into input values: scalars become
// their text form and nested objects or arrays are kept as JSON.
func InputsFromJSON(data []byte) (map[string]string, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]string{}, nil
	}
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var obj map[string]interface{}
	if err := dec.Decode(&obj); err != nil {
		return nil, fmt.Errorf("%w: body must be a JSON object: %v", ErrInvalidInput, err)
	}
	return InputsFromMap(obj), nil
}

// InputsFromMap converts decoded values (YAML or JSON) into input strings.
func InputsFromMap(obj map[string]interface{}) map[string]string {
	out := make(map[string]string, len(obj))
	for k, v := range obj {
		out[k] = inputValueString(v)
	}
	return out
}

func inputValueString(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case json.Number:
		return val.String()
	case bool, int, int64, float64:
		return fmt.Sprintf("%v", val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

// ParseInputFlags turns "key=value" CLI arguments into an input map.
func ParseInputFlags(flags []string) (map[string]string, error) {
	inputs := make(map[string]string, len(flags))
	for _, f := range flags {
		k, v, ok := strings.Cut(f, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("%w: expected --input key=value, got '%s'", ErrInvalidInput, f)
		}
		inputs[k] = v
	}
	return inputs, nil
}

type lenientInputsKey struct{}

// WithLenientInputs makes Run ignore undeclared inputs instead of rejecting
// them, for triggers whose payloads are shaped by an external system.
func WithLenientInputs(ctx context.Context) context.Context {
	return context.WithValue(ctx, lenientInputsKey{}, true)
}

// LenientInputs reports whether undeclared inputs should be ignored.
func LenientInputs(ctx context.Context) bool {
	v, _ := ctx.Value(lenientInputsKey{}).(bool)
	return v
}

// CheckInputs validates trigger-time inputs without running the workflow, so
// a trigger can reject a bad request before starting a background run.
func (w *Workflow) CheckInputs(provided map[string]string, strict bool) error {
	_, err := w.resolveInputs(provided, strict)
	return err
}
