package models

type TriggerType = string

const (
	TriggerManual    TriggerType = "manual"
	TriggerScheduled TriggerType = "cron"
	TriggerWebhook   TriggerType = "webhook"
)

type Trigger struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description"`
	Type        TriggerType `yaml:"type"`
	Schedule    string      `yaml:"schedule"`
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
