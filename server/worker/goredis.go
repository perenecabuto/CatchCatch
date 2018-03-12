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
	tasksQueue      = "catchcatch:worker:queue"
	processingQueue = "catchcatch:worker:processing"
	queueInterval   = time.Microsecond * 100
)

// GoredisWorkerManager is a simple manager implentation over go-redis
type GoredisWorkerManager struct {
	redis *redis.Client

	workersLock sync.RWMutex
	workers     map[string]Worker

	started      int32
	runningTasks int32
	stop         chan interface{}
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
				cmd := m.redis.RPopLPush(tasksQueue, processingQueue)
				m.redis.Process(cmd)
				err := cmd.Err()
				if err == redis.Nil {
					continue
				}
				if err != nil {
					log.Println("Run job redis err:", err)
					continue
				}
				go m.processTask(cmd.Val())
			}
		}
	}()
}

func (m *GoredisWorkerManager) processTask(encoded string) {
	id := gjson.Get(encoded, "id").String()
	w, exists := m.workers[id]
	if !exists {
		// TODO: requeue better
		log.Println("Ignoring: worker no found", id, m.workers)
		m.redis.LRem(processingQueue, -1, encoded).Err()
		m.redis.LPush(tasksQueue, encoded)
		time.Sleep(time.Second)
		return
	}

	atomic.AddInt32(&m.runningTasks, 1)
	params := make(map[string]string)
	for k, v := range gjson.Get(encoded, "params").Map() {
		params[k] = v.String()
	}

	err := w.Job(params)
	if err != nil {
		log.Println("Done job redis err:", err)

		return
	}
	err = m.redis.LRem(processingQueue, -1, encoded).Err()
	if err != nil {
		log.Printf("Job <%s> processTask - err: %s", id, err.Error())
	}
	atomic.AddInt32(&m.runningTasks, -1)
	log.Printf("Job <%s> done", id)
}

// Stop processing worker events
func (m *GoredisWorkerManager) Stop() {
	m.workersLock.Lock()
	defer m.workersLock.Unlock()
	if !m.Started() {
		return
	}
	timeout := time.NewTimer(time.Second * 20)
	defer timeout.Stop()

	m.waitForRemainingTasks(timeout)

	select {
	case m.stop <- true:
		return
	case <-timeout.C:
		log.Println("GoredisWorkerManager.Stop(): timeout")
		return
	}
}

func (m *GoredisWorkerManager) waitForRemainingTasks(timeout *time.Timer) {
	waitForTasksTicker := time.NewTicker(queueInterval * 2)
	defer waitForTasksTicker.Stop()
	for {
		runningTasks := atomic.LoadInt32(&m.runningTasks)
		select {
		case <-waitForTasksTicker.C:
			if runningTasks == int32(0) {
				return
			}
		case <-timeout.C:
			log.Printf("GoredisWorkerManager.Stop(): timed out with %d remaining tasks", m.runningTasks)
			return
		}
	}
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
	cmd := m.redis.LPush(tasksQueue, encoded)
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
	cmd := m.redis.LRange(processingQueue, 0, 100)
	err := m.redis.Process(cmd)
	encTasks, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(encTasks))
	for i, encoded := range encTasks {
		ids[i] = gjson.Get(encoded, "id").String()
	}
	return ids, nil
}

// Flush workers task queue
func (m *GoredisWorkerManager) Flush() error {
	cmd := m.redis.Del(tasksQueue)
	return m.redis.Process(cmd)
}
