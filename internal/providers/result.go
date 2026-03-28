package providers

// TaskResult holds the output and metadata from a provider execution.
type TaskResult struct {
	Output   string
	Metadata map[string]string
}
