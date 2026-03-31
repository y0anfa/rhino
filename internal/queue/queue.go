package queue

import (
	"context"
	"time"
)

type TaskMessage struct {
	ID           string                 `json:"id"`
	RunID        string                 `json:"run_id"`
	WorkflowName string                `json:"workflow_name"`
	TaskName     string                 `json:"task_name"`
	Provider     string                 `json:"provider"`
	Params       map[string]interface{} `json:"params"`
	MaxTries     int                    `json:"max_tries"`
	Timeout      time.Duration          `json:"timeout"`
	EnqueuedAt   time.Time              `json:"enqueued_at"`
}

type Queue interface {
	Enqueue(msg *TaskMessage) error
	Dequeue(ctx context.Context) (*TaskMessage, error)
	Ack(id string) error
	Nack(id string) error
	Len() int
	Close() error
}
