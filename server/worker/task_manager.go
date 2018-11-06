package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	JobHeartbeatInterval = time.Second * 5
)

type TaskManagerQueue interface {
	PollPending() (*Job, error)
	PollProcess() (*Job, error)
	EnqueuePending(*Job) error
	EnqueueToProcess(*Job) error
	GetProcessingJobByTaskAndParams(string, TaskParams) (*Job, error)
	RemoveFromProcessingQueue(*Job) error
	UpdateJobStatus(*Job)
}

type TaskManager struct {
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
		log.Println("Starting.... with worker manager")
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
				if job.Unique {
					processingJob, err := m.queue.GetProcessingJobByTaskAndParams(job.TaskID, job.Params)
					if err != nil {
						log.Println("[TaskManager]Start - can't check if job is already processing, reenqueing", err)
						m.queue.EnqueuePending(job)
						continue
					}
					if processingJob != nil {
						updateTimeout := time.Now().Add(JobHeartbeatInterval + time.Second)
						if processingJob.LastUpdate.Before(updateTimeout) {
							continue
						}
					}
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

// GetTaskByID returns a registered task. When no task is found it return an error
func (m *TaskManager) GetTaskByID(id string) (Task, error) {
	m.RLock()
	task, ok := m.tasks[id]
	m.RUnlock()
	if !ok {
		return nil, errors.Cause(fmt.Errorf("Task:%s is not registered", id))
	}
	return task, nil
}

func (m *TaskManager) processJob(ctx context.Context, job *Job) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	task, err := m.GetTaskByID(job.TaskID)
	if err != nil {
		m.queue.EnqueuePending(job)
		return errors.Wrapf(err, "can't get task:%s", job.TaskID)
	}

	go func() {
		ticker := time.NewTicker(JobHeartbeatInterval - (time.Millisecond * 10))
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
			case <-ticker.C:
				m.queue.UpdateJobStatus(job)
			}
		}
	}()

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
		runningJobs := atomic.LoadInt32(&m.runningJobs)
		select {
		case <-waitForJobsTicker.C:
			if runningJobs == int32(0) {
				return
			}
		case <-timeout.C:
			log.Printf("TaskManager.Stop(): timed out with %d remaining tasks", m.runningJobs)
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

// Run a task the worker
func (m *TaskManager) Run(t Task, params TaskParams) error {
	return m.run(t, params, false)
}

// RunUnique send a task the worker
// but it will be ignored if the worker is already running a task with the same parameters
func (m *TaskManager) RunUnique(t Task, params TaskParams) error {
	return m.run(t, params, true)
}

func (m *TaskManager) run(t Task, params TaskParams, unique bool) error {
	job := &Job{ID: uuid.New().String(), TaskID: t.ID(), Unique: unique, Params: params}
	err := m.queue.EnqueuePending(job)
	return errors.Cause(err)
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

// // BusyTasks ...
// func (m *TaskManager) BusyTasks() ([]string, error) {
// 	tasks, err := m.RunningJobs()
// 	if err != nil {
// 		return nil, err
// 	}
// 	ids := make(map[string]interface{})
// 	for _, t := range tasks {
// 		ids[t.TaskID] = nil
// 	}
// 	return funk.Keys(ids).([]string), nil
// }

// // RunningJobs return all running tasks
// func (m *TaskManager) RunningJobs() ([]Job, error) {
// 	encJobs := m.queue.ProcessingJobs()
// 	tasks := make([]Job, len(encJobs))
// 	for i, encoded := range encJobs {
// 		task := &Job{}
// 		json.Unmarshal([]byte(encoded), task)
// 		tasks[i] = *task
// 	}
// 	return tasks, nil
// }
