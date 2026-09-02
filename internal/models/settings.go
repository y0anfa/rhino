package models

import (
	"fmt"
	"time"
)

const (
	MaxTriesDefault       = 3
	TimeoutDefault        = "5s"
	RetryBackoffDefault   = "none"
	RetryBaseDelayDefault = "1s"
	RetryMaxDelayDefault  = "1m"
)

type Settings struct {
	MaxTries          int    `yaml:"max-tries" json:"max_tries"`
	Timeout           string `yaml:"timeout" json:"timeout"`
	RetryBackoff      string `yaml:"retry-backoff,omitempty" json:"retry_backoff,omitempty"`
	RetryBaseDelay    string `yaml:"retry-base-delay,omitempty" json:"retry_base_delay,omitempty"`
	RetryMaxDelay     string `yaml:"retry-max-delay,omitempty" json:"retry_max_delay,omitempty"`
	MaxConcurrentRuns int    `yaml:"max-concurrent-runs,omitempty" json:"max_concurrent_runs,omitempty"`
	MaxOutputSize     int    `yaml:"max-output-size,omitempty" json:"max_output_size,omitempty"`
}

// validateRetry checks the retry backoff strategy and its delays.
func (s *Settings) validateRetry() error {
	switch s.RetryBackoff {
	case "", "none", "linear", "exponential":
	default:
		return fmt.Errorf("invalid retry-backoff '%s': must be none, linear, or exponential", s.RetryBackoff)
	}
	if s.RetryBaseDelay != "" {
		if _, err := time.ParseDuration(s.RetryBaseDelay); err != nil {
			return fmt.Errorf("invalid retry-base-delay '%s': %w", s.RetryBaseDelay, err)
		}
	}
	if s.RetryMaxDelay != "" {
		if _, err := time.ParseDuration(s.RetryMaxDelay); err != nil {
			return fmt.Errorf("invalid retry-max-delay '%s': %w", s.RetryMaxDelay, err)
		}
	}
	return nil
}

func NewSettings(maxTries int, timeout string) *Settings {
	if maxTries < 0 {
		maxTries = MaxTriesDefault
	}
	if timeout == "" {
		timeout = TimeoutDefault
	}
	return &Settings{
		MaxTries:       maxTries,
		Timeout:        timeout,
		RetryBackoff:   RetryBackoffDefault,
		RetryBaseDelay: RetryBaseDelayDefault,
		RetryMaxDelay:  RetryMaxDelayDefault,
	}
}
