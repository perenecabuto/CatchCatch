package worker

import (
	"github.com/go-redis/redis"
)

// GoredisTaskManager is a simple manager implentation over go-redis
type GoredisTaskManager struct {
	*TaskManager
}

// NewGoredisTaskManager create a new GoredisTaskManager
func NewGoredisTaskManager(client redis.Cmdable, host string) Manager {
	queue := NewGoredisTaskQueue(client)
	manager := NewTaskManager(queue, host)
	return &GoredisTaskManager{TaskManager: manager}
}
