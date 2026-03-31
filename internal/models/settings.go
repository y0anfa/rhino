package models

const (
	MaxTriesDefault      = 3
	TimeoutDefault       = "5s"
	RetryBackoffDefault  = "none"
	RetryBaseDelayDefault = "1s"
	RetryMaxDelayDefault  = "1m"
)

type Settings struct {
	MaxTries           int    `yaml:"max-tries"`
	Timeout            string `yaml:"timeout"`
	RetryBackoff       string `yaml:"retry-backoff,omitempty"`
	RetryBaseDelay     string `yaml:"retry-base-delay,omitempty"`
	RetryMaxDelay      string `yaml:"retry-max-delay,omitempty"`
	MaxConcurrentRuns  int    `yaml:"max-concurrent-runs,omitempty"`
	MaxOutputSize      int    `yaml:"max-output-size,omitempty"`
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
