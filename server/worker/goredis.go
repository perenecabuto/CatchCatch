package worker

import (
	"log"
	"sync"

	"github.com/go-redis/redis"
)

// GoredisWorkerManager is a simple manager implentation over go-redis
type GoredisWorkerManager struct {
	redis *redis.Client

	workers     map[string]Worker
	workersLock sync.RWMutex
}

// NewGoredisWorkerManager create a new GoredisWorkerManager
func NewGoredisWorkerManager(client *redis.Client) Manager {
	return &GoredisWorkerManager{redis: client, workers: make(map[string]Worker)}
}

// Start listening tasks events
func (m *GoredisWorkerManager) Start() {
	log.Println("Starting.... with redis:", m.redis)
}

// Stop ... ???
func (m *GoredisWorkerManager) Stop() {
	panic("not implemented")
}

// Add add worker to this manager
func (m *GoredisWorkerManager) Add(w Worker) {
	m.workersLock.Lock()
	m.workers[w.ID()] = w
	m.workersLock.Unlock()
}

// Run a task into the worker
func (m *GoredisWorkerManager) Run(w Worker, params map[string]string) error {
}

// WorkersIDs return managed workers ids
func (m *GoredisWorkerManager) WorkersIDs() []string {
	ids := make([]string, len(m.workers))
	m.workersLock.RLock()
	count := 0
	for _, w := range m.workers {
		ids[count] = w.ID()
		count++
	}
	m.workersLock.RUnlock()
	return ids
}
