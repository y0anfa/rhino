package workflow

import (
	"fmt"
)

type TriggerType = string

const (
	TriggerManual    TriggerType = "manual"
	TriggerScheduled TriggerType = "scheduled"
	TriggerWebhook   TriggerType = "webhook"
)

type Trigger struct {
	Name        string 	    `yaml:"name"`
	Description string      `yaml:"description"`
	Type        TriggerType `yaml:"type"`
	Schedule    string 	    `yaml:"schedule"`
}

func NewTrigger(name string, desc string, triggertype TriggerType, schedule string) (*Trigger, error) {
	switch triggertype {
	case TriggerManual:
		return &Trigger{Name: name, Description: desc, Type: triggertype}, nil
	case TriggerScheduled:
		return &Trigger{Name: name, Description: desc, Type: triggertype, Schedule: schedule}, nil
	default:
		return nil, fmt.Errorf("invalid trigger type: %s", triggertype)
	}
}
