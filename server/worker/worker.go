package worker

import (
	"context"
	"strings"
)

// Manager for workers and its tasks
type Manager interface {
	Start(ctx context.Context)
	Stop()
	Started() bool
	Add(w Worker)

	Run(w Worker, params map[string]string) error
	RunUnique(w Worker, params map[string]string) error

	BusyWorkers() ([]string, error)
	Flush() error
}

// Worker runs tasks
type Worker interface {
	ID() string
	Job(params map[string]string) error
}

// Task represents a worker job
type Task struct {
	ID       string
	WorkerID string
	Unique   bool
	Params   map[string]string
}

// LockName return a unique lock name for this task
func (t Task) LockName() string {
	return strings.Join([]string{tasksQueue, t.WorkerID, "lock"}, ":")
}
