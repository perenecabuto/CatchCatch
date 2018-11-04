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
	funk "github.com/thoas/go-funk"
)

const (
	tasksQueue      = "catchcatch:worker:queue"
	processingQueue = "catchcatch:worker:processing"
)

var QueuePollInterval = time.Second / 4

// GoredisTaskManager is a simple manager implentation over go-redis
type GoredisTaskManager struct {
	redis redis.Cmdable

	tasksLock sync.RWMutex
	tasks     map[string]Task

	started     int32
	runningJobs int32
	stop        chan interface{}
}

// NewGoredisTaskManager create a new GoredisTaskManager
func NewGoredisTaskManager(client redis.Cmdable) Manager {
	return &GoredisTaskManager{redis: client,
		tasks: make(map[string]Task),
		stop:  make(chan interface{}, 1)}
}

// Started return if worker is started
func (m *GoredisTaskManager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

// Start listening tasks events
func (m *GoredisTaskManager) Start(ctx context.Context) {
	go func() {
		log.Println("Starting.... with redis:", m.redis)
		atomic.StoreInt32(&m.started, 1)

		wCtx, cancel := context.WithCancel(ctx)

		timer := time.NewTimer(QueuePollInterval)
		for {
			select {
			case <-wCtx.Done():
				go m.Stop()
			case <-m.stop:
				cancel()
				atomic.StoreInt32(&m.started, 0)
				return
			case <-timer.C:

				timer.Reset(QueuePollInterval)
				cmd := m.redis.RPopLPush(tasksQueue, processingQueue)
				err := cmd.Err()
				if err == redis.Nil {
					continue
				}
				if err != nil {
					log.Println("Run job redis err:", err)
					continue
				}

				go m.processJob(wCtx, cmd.Val())
			}
		}
	}()
}

func (m *GoredisTaskManager) processJob(ctx context.Context, encoded string) {
	task := &Job{}
	json.Unmarshal([]byte(encoded), task)
	w, exists := m.tasks[task.TaskID]
	if !exists {
		// FIXME: requeue better
		log.Println("Ignoring task: worker not found", task.TaskID, m.tasks)
		m.redis.LRem(processingQueue, -1, encoded).Err()
		m.redis.LPush(tasksQueue, encoded)
		time.Sleep(time.Second)
		return
	}

	if task.Unique {
		lock := task.LockName()
		// TODO: implement heartbeat
		cmd := m.redis.SetNX(lock, task.TaskID, time.Second*30)
		if !cmd.Val() {
			log.Println("Unique task already running:", lock)
			m.redis.LRem(processingQueue, -1, encoded).Err()
			return
		}
		log.Println("Run unique task:", lock)
		defer m.redis.Del(lock)
	}

	atomic.AddInt32(&m.runningJobs, 1)

	err := w.Run(ctx, task.Params)
	if err != nil {
		log.Println("Done job redis err:", err)
		return
	}
	err = m.redis.LRem(processingQueue, -1, encoded).Err()
	if err != nil {
		log.Printf("Run <%s> processJob - err: %s", task.TaskID, err.Error())
	}
	atomic.AddInt32(&m.runningJobs, -1)
	log.Printf("Run <%s> done", task.TaskID)
}

// Stop processing worker events
func (m *GoredisTaskManager) Stop() {
	m.tasksLock.Lock()
	defer m.tasksLock.Unlock()
	if !m.Started() {
		return
	}
	timeout := time.NewTimer(time.Second * 20)
	defer timeout.Stop()

	m.waitForRemainingJobs(timeout)

	select {
	case m.stop <- true:
		return
	case <-timeout.C:
		log.Println("GoredisTaskManager.Stop(): timeout")
		return
	}
}

func (m *GoredisTaskManager) waitForRemainingJobs(timeout *time.Timer) {

	waitForJobsTicker := time.NewTicker(QueuePollInterval * 2)
	defer waitForJobsTicker.Stop()
	for {
		runningJobs := atomic.LoadInt32(&m.runningJobs)
		select {
		case <-waitForJobsTicker.C:
			if runningJobs == int32(0) {
				return
			}
		case <-timeout.C:
			log.Printf("GoredisTaskManager.Stop(): timed out with %d remaining tasks", m.runningJobs)
			return
		}
	}
}

// Add add worker to this manager
func (m *GoredisTaskManager) Add(w Task) {
	m.tasksLock.Lock()
	m.tasks[w.ID()] = w
	m.tasksLock.Unlock()
}

// Run a task the worker
func (m *GoredisTaskManager) Run(w Task, params TaskParams) error {
	return m.run(w, params, false)
}

// RunUnique send a task the worker
// but it will be ignored if the worker is already running a task with the same parameters
func (m *GoredisTaskManager) RunUnique(w Task, params TaskParams) error {
	return m.run(w, params, true)
}

func (m *GoredisTaskManager) run(w Task, params TaskParams, unique bool) error {
	task := &Job{ID: uuid.New().String(), TaskID: w.ID(), Unique: unique, Params: params}
	cmd := m.redis.Exists(task.LockName())
	exists, err := cmd.Val() == 1, cmd.Err()
	if err != nil {
		return err
	}
	if exists {
		// log.Printf("Skiping worker<%s> is locked:", task.LockName())
		return nil
	}

	encoded, _ := json.Marshal(task)
	return m.redis.LPush(tasksQueue, encoded).Err()
}

// TasksIDs return managed tasks ids
func (m *GoredisTaskManager) TasksIDs() []string {
	ids := make([]string, len(m.tasks))
	m.tasksLock.RLock()
	count := 0
	for _, w := range m.tasks {
		ids[count] = w.ID()
		count++
	}
	m.tasksLock.RUnlock()
	return ids
}

// BusyTasks ...
func (m *GoredisTaskManager) BusyTasks() ([]string, error) {
	tasks, err := m.RunningJobs()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]interface{})
	for _, t := range tasks {
		ids[t.TaskID] = nil
	}
	return funk.Keys(ids).([]string), nil
}

// RunningJobs return all running tasks
func (m *GoredisTaskManager) RunningJobs() ([]Job, error) {
	cmd := m.redis.LRange(processingQueue, 0, 100)
	encJobs, err := cmd.Result()
	if err != nil {
		return nil, err
	}
	tasks := make([]Job, len(encJobs))
	for i, encoded := range encJobs {
		task := &Job{}
		json.Unmarshal([]byte(encoded), task)
		tasks[i] = *task
	}
	return tasks, nil
}

// Flush tasks task queue
func (m *GoredisTaskManager) Flush() error {
	return m.redis.Del(tasksQueue).Err()
}
