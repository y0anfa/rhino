/* Create the generic provider interface.
A provider is a plugin that integrates a third-party service with this application. */
package providers

import (
	"fmt"
)

// Provider is the interface that all providers must implement.
type Provider interface {
	// Name returns the name of the provider.
	Name() string
	// Validate validates the provider arguments.
	Validate(args map[string]interface{}) error
	// Run runs the provider with the given arguments and returns the result.
	Run(args map[string]interface{}) (*TaskResult, error)
}

// Create a map of providers.
var providers = make(map[string]Provider)

// Register registers a provider.
func Register(provider Provider) {
	providers[provider.Name()] = provider
}

// Get gets a provider by name.
func Get(name string) (Provider, error) {
	provider, ok := providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return provider, nil
}

// List lists all registered providers.
func List() []string {
	var list []string
	for name := range providers {
		list = append(list, name)
	}
	return list
}


