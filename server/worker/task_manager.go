package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

var (
	// JobHeartbeatInterval is the interval to notify job running
	JobHeartbeatInterval = time.Second * 5
	QueuePollInterval    = time.Second / 4

	ErrJobAlreadySet = errors.New("job already set")
)

// TaskManagerQueue manager jobs queue
type TaskManagerQueue interface {
	PollPending() (*Job, error)
	PollProcess() (*Job, error)
	EnqueuePending(*Job) error
	EnqueueToProcess(*Job) error
	RemoveFromProcessingQueue(*Job) error

	JobsOnProcessQueue() ([]*Job, error)

	IsJobAlreadyRunning(*Job) (bool, error)
	SetJobRunning(*Job, time.Duration) (bool, error)
	SetJobDone(*Job) error
	HeartbeatJob(*Job, time.Duration) error
}

type TaskManager struct {
	host        string
	queue       TaskManagerQueue
	tasks       map[string]Task
	started     int32
	runningJobs int32
	stop        chan interface{}

	sync.RWMutex
}

// NewTaskManager creates a TaskManager
func NewTaskManager(e TaskManagerQueue) *TaskManager {
	return &TaskManager{queue: e,
		tasks: make(map[string]Task),
		stop:  make(chan interface{}, 1),
func NewTaskManager(e TaskManagerQueue, host string) *TaskManager {
	return &TaskManager{host: host, queue: e,
		tasks:       make(map[string]Task),
		stop:        make(chan interface{}, 1),
	}
}

// Started return if worker is started
func (m *TaskManager) Started() bool {
	return atomic.LoadInt32(&m.started) == 1
}

// GetTaskByID returns a registered task. When no task is found it return an error
func (m *TaskManager) GetTaskByID(id string) (Task, error) {
	m.RLock()
	task, ok := m.tasks[id]
	m.RUnlock()
	if !ok {
		return nil, errors.Cause(fmt.Errorf("task:%s is not registered", id))
	}
	return task, nil
}

// Start listening tasks events
func (m *TaskManager) Start(ctx context.Context) {
	go func() {
		log.Println("Starting worker manager on:", m.host)
		atomic.StoreInt32(&m.started, 1)

		wCtx, cancel := context.WithCancel(ctx)
		pendingQueueInterval := time.NewTimer(QueuePollInterval)
		processingQueueInterval := time.NewTimer(QueuePollInterval)
		for {
			select {
			case <-wCtx.Done():
				m.Stop()
			case <-m.stop:
				cancel()
				return
			case <-processingQueueInterval.C:
				processingQueueInterval.Reset(QueuePollInterval)
				job, err := m.queue.PollProcess()
				if err != nil {
					log.Println("[TaskManager]Start - can't poll", err)
					continue
				}
				if job == nil {
					continue
				}
				job.Host = host
				aquired, err := m.queue.SetJobRunning(job, JobHeartbeatInterval)
				if err != nil {
					log.Println("[TaskManager]Start - can't set job running from processing queue, reenqueing", err)
					continue
				}
				if !aquired {
					log.Println("[TaskManager]Start - job is already running on processing queue - skipping")
					continue
				}
				go m.processJob(wCtx, job)

			case <-pendingQueueInterval.C:
				pendingQueueInterval.Reset(QueuePollInterval)
				job, err := m.queue.PollPending()
				if err != nil {
					log.Println("[TaskManager]Start - can't poll", err)
					continue
				}
				if job == nil {
					continue
				}
				running, err := m.queue.IsJobAlreadyRunning(job)
				if err != nil {
					log.Println("[TaskManager]Start - can't check if job is already running, reenqueing", err)
					m.queue.EnqueuePending(job)
					continue
				}
				if running {
					log.Println("[TaskManager]Start - job is already running, skipping")
					continue
				}
				err = m.queue.EnqueueToProcess(job)
				if err != nil {
					log.Println("[TaskManager]Start - can't add job to processing queue, reenqueing", err)
					m.queue.EnqueuePending(job)
					continue
				}
			}
		}
	}()
}

func (m *TaskManager) processJob(ctx context.Context, job *Job) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	task, err := m.GetTaskByID(job.TaskID)
	if err != nil {
		err := m.queue.EnqueuePending(job)
		if err != nil {
			return errors.Wrapf(err, "can't reenqueue job:%s, skipping", job.ID)
		}
		m.queue.RemoveFromProcessingQueue(job)
		return errors.Wrapf(err, "can't get task:%s", job.TaskID)
	}

	atomic.AddInt32(&m.runningJobs, 1)

	err = task.Run(ctx, job.Params)
	err = errors.Wrapf(err, "error running job:%s", job.ID)

	removeErr := m.queue.RemoveFromProcessingQueue(job)
	errors.Wrapf(removeErr, "can't set DONE to job:%s", job.ID)

	atomic.AddInt32(&m.runningJobs, -1)
	log.Printf("task:%s:job:%s:done", job.TaskID, job.ID)

	return err
}

// Stop processing worker events
func (m *TaskManager) Stop() {
	if !m.Started() {
		return
	}
	atomic.StoreInt32(&m.started, 0)

	timeout := time.NewTimer(time.Second * 20)
	defer timeout.Stop()

	m.waitForRemainingJobs(timeout)

	select {
	case m.stop <- true:
		return
	case <-timeout.C:
		log.Println("TaskManager.Stop(): timeout")
		return
	}
}

func (m *TaskManager) waitForRemainingJobs(timeout *time.Timer) {
	waitForJobsTicker := time.NewTicker(QueuePollInterval * 2)
	defer waitForJobsTicker.Stop()
	for {
		select {
		case <-waitForJobsTicker.C:
			if m.RunningJobs() == 0 {
				return
			}
		case <-timeout.C:
			log.Printf("TaskManager.Stop(): timed out with %d remaining tasks", m.RunningJobs())
			return
		}
	}
}

// Add add worker to this manager
func (m *TaskManager) Add(t Task) {
	m.Lock()
	m.tasks[t.ID()] = t
	m.Unlock()
}

// TaskIDs return managed tasks ids
func (m *TaskManager) TaskIDs() []string {
	ids := make([]string, len(m.tasks))
	count := 0

	m.RLock()
	for id := range m.tasks {
		ids[count] = id
		count++
	}
	m.RUnlock()
	return ids
}

// RunningJobs return total of jobs running on this manager
func (m *TaskManager) RunningJobs() int {
	runningJobs := atomic.LoadInt32(&m.runningJobs)
	return int(runningJobs)
}
