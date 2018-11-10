package worker

import (
	"context"
	"time"
)

// Manager for workers and its tasks
type Manager interface {
	Start(ctx context.Context)
	Stop()
	Started() bool
	Add(w Task)

	Run(w Task, params TaskParams) error
	RunUnique(w Task, params TaskParams) error

	Flush() error
}

// TaskParams is map of task parameters
type TaskParams map[string]interface{}

// Task runs tasks
type Task interface {
	ID() string
	Run(ctx context.Context, params TaskParams) error
}

// Job represents a worker job
type Job struct {
	ID         string
	TaskID     string
	Params     TaskParams
	Host       string
	LastUpdate time.Time
}
