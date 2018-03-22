package worker

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis"
	"github.com/google/uuid"
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
	task := &Task{}
	json.Unmarshal([]byte(encoded), task)
	w, exists := m.workers[task.WorkerID]
	if !exists {
		// FIXME: requeue better
		log.Println("Ignoring task: worker not found", task.WorkerID, m.workers)
		m.redis.LRem(processingQueue, -1, encoded).Err()
		m.redis.LPush(tasksQueue, encoded)
		time.Sleep(time.Second)
		return
	}

	if task.Unique {
		lock := task.LockName()
		cmd := redis.NewStringCmd("SET", lock, task.WorkerID, "PX", 30000, "NX")
		m.redis.Process(cmd)
		if cmd.Val() != "OK" {
			log.Println("Unique task already running:", lock)
			m.redis.LRem(processingQueue, -1, encoded).Err()
			return
		}
		log.Println("Run unique task:", lock)
		defer m.redis.Del(lock)
	}

	atomic.AddInt32(&m.runningTasks, 1)

	err := w.Job(task.Params)
	if err != nil {
		log.Println("Done job redis err:", err)
		return
	}
	err = m.redis.LRem(processingQueue, -1, encoded).Err()
	if err != nil {
		log.Printf("Job <%s> processTask - err: %s", task.WorkerID, err.Error())
	}
	atomic.AddInt32(&m.runningTasks, -1)
	log.Printf("Job <%s> done", task.WorkerID)
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

// Run a task the worker
func (m *GoredisWorkerManager) Run(w Worker, params map[string]string) error {
	return m.run(w, params, false)
}

// RunUnique send a task the worker
// but it will be ignored if the worker is already running a task with the same parameters
func (m *GoredisWorkerManager) RunUnique(w Worker, params map[string]string) error {
	return m.run(w, params, true)
}

func (m *GoredisWorkerManager) run(w Worker, params map[string]string, unique bool) error {
	task := &Task{ID: uuid.New().String(), WorkerID: w.ID(), Unique: unique, Params: params}
	cmd := m.redis.Exists(task.LockName())
	exists, err := cmd.Val() == 1, cmd.Err()
	if err != nil {
		return err
	}
	if exists {
		log.Printf("Skiping worker<%s> is locked:", task.LockName())
		return nil
	}

	encoded, _ := json.Marshal(task)
	return m.redis.LPush(tasksQueue, encoded).Err()
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

// BusyWorkers ...
func (m *GoredisWorkerManager) BusyWorkers() ([]string, error) {
	tasks, err := m.RunningTasks()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]interface{})
	for _, t := range tasks {
		ids[t.WorkerID] = nil
	}
	return funk.Keys(ids).([]string), nil
}

// RunningTasks return all running tasks
func (m *GoredisWorkerManager) RunningTasks() ([]Task, error) {
	cmd := m.redis.LRange(processingQueue, 0, 100)
	encTasks, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	tasks := make([]Task, len(encTasks))
	for i, encoded := range encTasks {
		task := &Task{}
		json.Unmarshal([]byte(encoded), task)
		tasks[i] = *task
	}
	return tasks, nil
}

// Flush workers task queue
func (m *GoredisWorkerManager) Flush() error {
	return m.redis.Del(tasksQueue).Err()
}
