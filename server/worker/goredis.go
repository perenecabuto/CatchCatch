package worker

import (
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	queue         = "catchcatch:worker:queue"
	queueInterval = time.Microsecond * 100
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
	go func() {
		log.Println("Starting.... with redis:", m.redis)

		ticker := time.NewTicker(queueInterval)
		for {
			select {
			case <-ticker.C:
				cmd := m.redis.RPop(queue)
				m.redis.Process(cmd)
				err := cmd.Err()
				if err == redis.Nil {
					continue
				}
				if err != nil {
					log.Println(err)
					continue
				}
				encoded := cmd.String()
				id := gjson.Get(encoded, "id").String()
				w, exists := m.workers[id]
				if !exists {
					continue
				}
				params := make(map[string]string)
				for k, v := range gjson.Get(encoded, "params").Map() {
					params[k] = v.String()
				}
				w.Job(params)
			}
		}
	}()
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
	encoded, _ := sjson.Set("", "id", w.ID())
	encoded, err := sjson.Set(encoded, "params", params)
	if err != nil {
		return err
	}
	cmd := m.redis.LPush(queue, encoded)
	return m.redis.Process(cmd)
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
