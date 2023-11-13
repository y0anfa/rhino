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
	// Run runs the provider with the given arguments.
	Run(args []string) error
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


