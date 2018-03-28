package worker

import (
	"context"
	"strings"

	"github.com/tidwall/sjson"
)

// Manager for workers and its tasks
type Manager interface {
	Start(ctx context.Context)
	Stop()
	Started() bool
	Add(w Worker)

	Run(w Worker, params TaskParams) error
	RunUnique(w Worker, params TaskParams) error

	BusyWorkers() ([]string, error)
	RunningTasks() ([]Task, error)
	Flush() error
}

// Worker runs tasks
type Worker interface {
	ID() string
	Run(ctx context.Context, params TaskParams) error
}

// TaskParams is map of task parameters
type TaskParams map[string]interface{}

// Task represents a worker job
type Task struct {
	ID       string
	WorkerID string
	Unique   bool
	Params   TaskParams
}

// LockName return a unique lock name for this task
func (t Task) LockName() string {
	params, _ := sjson.Set("", "params", t.Params)
	return strings.Join([]string{tasksQueue, t.WorkerID, params, "lock"}, ":")
}
