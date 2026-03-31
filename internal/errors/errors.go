package errors

import "fmt"

type WorkflowNotFoundError struct {
	Name       string
	Suggestion string
}

func (e *WorkflowNotFoundError) Error() string {
	msg := fmt.Sprintf("workflow '%s' not found", e.Name)
	if e.Suggestion != "" {
		msg += fmt.Sprintf(". Did you mean '%s'?", e.Suggestion)
	}
	return msg
}

type ProviderNotFoundError struct {
	Name string
}

func (e *ProviderNotFoundError) Error() string {
	return fmt.Sprintf("provider '%s' not found", e.Name)
}

type ValidationError struct {
	Entity  string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Entity, e.Message)
}

type TimeoutError struct {
	TaskName string
	Duration string
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("task '%s' timed out after %s", e.TaskName, e.Duration)
}

type SecretNotFoundError struct {
	Key string
}

func (e *SecretNotFoundError) Error() string {
	return fmt.Sprintf("secret '%s' not found", e.Key)
}

// Levenshtein computes edit distance between two strings for suggestions.
func Levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// FindClosest finds the closest match from candidates using Levenshtein distance.
func FindClosest(input string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	best := ""
	bestDist := len(input) + 1
	for _, c := range candidates {
		d := Levenshtein(input, c)
		if d < bestDist && d <= len(input)/2+1 {
			bestDist = d
			best = c
		}
	}
	return best
}
