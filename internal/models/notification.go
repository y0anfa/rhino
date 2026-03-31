package models

type NotificationConfig struct {
	OnSuccess []NotificationChannel `yaml:"on-success,omitempty"`
	OnFailure []NotificationChannel `yaml:"on-failure,omitempty"`
}

type NotificationChannel struct {
	Provider string                 `yaml:"provider"`
	Params   map[string]interface{} `yaml:"params"`
}
