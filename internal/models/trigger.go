package models

type TriggerType = string

const (
	TriggerManual    TriggerType = "manual"
	TriggerScheduled TriggerType = "cron"
	TriggerWebhook   TriggerType = "webhook"
	TriggerWatch     TriggerType = "watch"
)

type Trigger struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Type        TriggerType `yaml:"type"`
	Schedule    string      `yaml:"schedule,omitempty"`
	WatchPath    string   `yaml:"watch-path,omitempty"`
	WatchPattern string   `yaml:"watch-pattern,omitempty"`
	WatchEvents  []string `yaml:"watch-events,omitempty"`
	Debounce    string      `yaml:"debounce,omitempty"`
}

func NewTrigger(name string, desc string, triggertype TriggerType, schedule string) *Trigger {
	switch triggertype {
	case TriggerManual:
		return &Trigger{Name: name, Description: desc, Type: triggertype}
	case TriggerScheduled:
		return &Trigger{Name: name, Description: desc, Type: triggertype, Schedule: schedule}
	case TriggerWebhook:
		return &Trigger{Name: name, Description: desc, Type: triggertype}
	default:
		return nil
	}
}
