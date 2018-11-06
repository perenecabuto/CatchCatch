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
	JobHeartbeatInterval = time.Second * 5

	ErrJobAlreadySet = errors.New("job already set")
)

type TaskManagerQueue interface {
	PollPending() (*Job, error)
	PollProcess() (*Job, error)
	EnqueuePending(*Job) error
	EnqueueToProcess(*Job) error
	RemoveFromProcessingQueue(*Job) error
	JobsOnProcessQueue() ([]*Job, error)

	SetJob(*Job) error
	GetJobByID(string) (*Job, error)
	RemoveJob(*Job) error
	SetJobLock(*Job, time.Duration) (bool, error)
	UpdateJobLock(*Job, time.Duration) (bool, error)
	HeartbeatJob(*Job, time.Duration) error
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
				processingJob, err := m.queue.GetJobByID(job.ID)
				if err != nil {
					log.Println("[TaskManager]Start - can't get job by id, reenqueing", err)
					continue
				}
				if processingJob != nil && processingJob.IsUpdatedToInterval(JobHeartbeatInterval+time.Second) {
					continue
				}
				err = m.queue.SetJob(job)
				if err != nil {
					log.Println("[TaskManager]Start - can't set job on processing queue, reenqueing", err)
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
					aquired, err := m.queue.SetJobLock(job, JobHeartbeatInterval*2)
					if err != nil {
						log.Println("[TaskManager]Start - can't check if job is already processing, reenqueing", err)
						m.queue.EnqueuePending(job)
						continue
					}
					if !aquired {
						continue
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
// RunningJobs return total of jobs running on this manager
func (m *TaskManager) RunningJobs() int {
	runningJobs := atomic.LoadInt32(&m.runningJobs)
	return int(runningJobs)
}


}
