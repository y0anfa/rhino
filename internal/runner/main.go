package runner

import (
	"github.com/y0anfa/rhino/internal/logger"
	"github.com/y0anfa/rhino/internal/models"
)

func init() {
	RunnerManager := NewRunnerManager()

	workflows, err := models.LoadWorkflows()
	if err != nil {
		logger.Error("error while listing workflows: ", err)
	}

	for _, w := range workflows {
		switch w.Trigger.Type {
		case models.TriggerScheduled:
			RunnerManager.AddRunner(&CronRunner{Workflow: w})
		case models.TriggerWebhook:
			RunnerManager.AddRunner(&WebhookRunner{Workflow: w})
		default:
			logger.Error("unknown trigger type: ", w.Trigger.Name)
		}
	}
}

