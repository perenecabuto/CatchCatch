package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
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

	workersLock sync.RWMutex
	workers     map[string]Worker

	started int32
	stop    chan interface{}
}

// NewGoredisWorkerManager create a new GoredisWorkerManager
func NewGoredisWorkerManager(client *redis.Client) Manager {
	return &GoredisWorkerManager{redis: client, workers: make(map[string]Worker), stop: make(chan interface{}, 1)}
}

// Started return if worker is started
func (m *GoredisWorkerManager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

// Start listening tasks events
func (m *GoredisWorkerManager) Start(ctx context.Context) {
	go func() {
		log.Println("Starting.... with redis:", m.redis)
		atomic.StoreInt32(&m.started, 1)

		ticker := time.NewTicker(queueInterval)
		for {
			select {
			case <-ctx.Done():
				go m.Stop()
			case <-m.stop:
				atomic.StoreInt32(&m.started, 0)
				return
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
	m.stop <- true
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

// BusyWorkers return a list of busy workers
func (m *GoredisWorkerManager) BusyWorkers() ([]string, error) {
	return ids, nil
}

// Flush workers task queue
func (m *GoredisWorkerManager) Flush() error {
	cmd := m.redis.Del(queue)
	return m.redis.Process(cmd)
}
