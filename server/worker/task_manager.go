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
	PollProcess(updatedUntil time.Time) (*Job, error)
	EnqueuePending(*Job) error
	EnqueueToProcess(*Job) error
	RemoveFromProcessingQueue(*Job) error
	SetJobRunning(job *Job, host string, lockDuration time.Duration) (bool, error)
	HeartbeatJob(job *Job) error

	IsJobOnProcessQueue(job *Job) (bool, error)
	JobsOnProcessQueue() ([]*Job, error)

	Flush() error
}

type TaskManager struct {
	host        string
	queue       TaskManagerQueue
	started     int32
	tasks       map[string]Task
	runningJobs map[string]string
	stop        chan interface{}

	sync.RWMutex
}

// NewTaskManager creates a TaskManager
func NewTaskManager(e TaskManagerQueue, host string) *TaskManager {
	return &TaskManager{host: host, queue: e,
		runningJobs: make(map[string]string),
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
				updatedBefore := time.Now().Add(-(JobHeartbeatInterval * 2))
				job, err := m.queue.PollProcess(updatedBefore)
				if err != nil {
					log.Println("[TaskManager]Start - can't poll", err)
					continue
				}
				if job == nil {
					continue
				}
				aquired, err := m.queue.SetJobRunning(job, m.host, JobHeartbeatInterval*2)
				if err != nil {
					log.Println(errors.Wrapf(err, "can't set job:%s:task:%s running", job.ID, job.TaskID))
					continue
				}
				if !aquired {
					// log.Println(errors.New("skipping, job already running"))
					continue
				}
				go m.processJob(wCtx, job)
				processingQueueInterval.Reset(time.Second)

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
				skip, err := m.queue.IsJobOnProcessQueue(job)
				if err != nil {
					log.Println("[TaskManager]Start - can't check job on processing queue, reenqueing", err)
					m.queue.EnqueuePending(job)
					continue
				}
				if skip {
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

	m.Lock()
	m.runningJobs[job.ID] = job.TaskID
	m.Unlock()
	log.Println("Starting JOB!!!", job.ID, m.RunningJobs())

	go func() {
		ticker := time.NewTicker(JobHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.queue.HeartbeatJob(job)
			}
		}
	}()

	err = task.Run(ctx, job.Params)
	err = errors.Wrapf(err, "error running job:%s", job.ID)

	removeErr := m.queue.RemoveFromProcessingQueue(job)
	errors.Wrapf(removeErr, "can't set DONE to job:%s", job.ID)

	m.Lock()
	delete(m.runningJobs, job.ID)
	m.Unlock()
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
			if len(m.RunningJobs()) == 0 {
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

// TasksID return managed tasks ids
func (m *TaskManager) TasksID() []string {
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
func (m *TaskManager) RunningJobs() []string {
	jobIDs := make([]string, 0)
	m.RLock()
	for id := range m.runningJobs {
		jobIDs = append(jobIDs, id)
	}
	m.RUnlock()
	return jobIDs
}
