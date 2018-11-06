package worker_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/perenecabuto/CatchCatch/server/worker/mocks"
)

func TestTaskManagerRegisterTasks(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	manager := worker.NewTaskManager(queue)

	task1 := &mocks.Task{}
	task1.On("ID").Return("task-1")
	task2 := &mocks.Task{}
	task2.On("ID").Return("task-2")

	manager.Add(task1)
	manager.Add(task2)

	actual := manager.TaskIDs()
	assert.Contains(t, actual, "task-1")
	assert.Contains(t, actual, "task-2")
}

func TestTaskManagerGetTasksByID(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	manager := worker.NewTaskManager(queue)

	task1 := &mocks.Task{}
	task1.On("ID").Return("task-1")

	manager.Add(task1)

	actual, err := manager.GetTaskByID("task-1")
	require.NoError(t, err)
	assert.Equal(t, actual, task1)
}

func TestTaskManagerGetTasksByIDReturnAnErrorWhenTaskIsNotRegistered(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	manager := worker.NewTaskManager(queue)

	task, err := manager.GetTaskByID("task-1")
	assert.Nil(t, task)
	assert.Error(t, err)
}

func TestTaskManagerStart(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(nil, nil)
	manager := worker.NewTaskManager(queue)
	require.False(t, manager.Started())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return manager.Started()
	}).Should(gomega.BeTrue())
}

func TestTaskManagerStopWhenContextIsDone(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(nil, nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return manager.Started()
	}).Should(gomega.BeTrue())

	cancel()
	gomega.Eventually(func() bool {
		return manager.Started()
	}).Should(gomega.BeFalse())
}

func TestTaskManagerDoNotAddJobToProcessQueueWhenItFailsToPoll(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	queue.On("PollPending").Return(nil, errors.New("an error"))
	queue.On("PollProcess").Return(nil, nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "PollPending")
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)

	queue.AssertNotCalled(t, "EnqueueToProcess", mock.Anything)
}

func TestTaskManagerReenqueueUniqueJobWhenItFailsToCheckIfItIsLocked(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	job := &worker.Job{
		ID:         "reenqueued",
		Unique:     true,
		TaskID:     "task",
		LastUpdate: time.Now(),
	}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess").Return(nil, nil)
	queue.On("SetJobLock", job, mock.Anything).Return(false, errors.New("fail"))
	queue.On("EnqueuePending", mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJobLock", mock.Anything, mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)

	queue.AssertCalled(t, "EnqueuePending", job)
}

func TestTaskManagerDiscardUniqueJobWhenLockIsAlreadyAquired(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	job := &worker.Job{
		ID:         "unique",
		TaskID:     "task",
		Unique:     true,
		LastUpdate: time.Now(),
	}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess").Return(nil, nil)
	queue.On("SetJobLock", job, mock.Anything).Return(false, nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJobLock", mock.Anything, mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertNotCalled(t, "EnqueueToProcess", mock.Anything)
}

func TestTaskManagerEnqueueJobsToProcess(t *testing.T) {
	jobs := []*worker.Job{
		{ID: "unique", TaskID: "task", Unique: true},
		{ID: "ordinary", TaskID: "task"},
	}
	for _, job := range jobs {
		queue := &mocks.TaskManagerQueue{}
		queue.On("PollProcess").Return(nil, nil)
		queue.On("PollPending").Return(job, nil)
		queue.On("SetJobLock", job, mock.Anything).Return(true, nil)
		queue.On("EnqueueToProcess", mock.Anything).Return(nil)

		manager := worker.NewTaskManager(queue)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager.Start(ctx)

		gomega.RegisterTestingT(t)
		gomega.Eventually(func() bool {
			return queue.AssertCalled(&testing.T{}, "PollPending")
		}).Should(gomega.BeTrue())

		time.Sleep(time.Millisecond * 10)
		queue.AssertCalled(t, "EnqueueToProcess", job)
	}
}

func TestTaskManagerReenqueueToPendingWhenFailToEnqueueToProcess(t *testing.T) {
	jobs := []*worker.Job{
		{ID: "unique", TaskID: "task", Unique: true},
		{ID: "ordinary", TaskID: "task"},
	}
	for _, job := range jobs {
		queue := &mocks.TaskManagerQueue{}
		queue.On("PollProcess").Return(nil, nil)
		queue.On("PollPending").Return(job, nil)
		queue.On("SetJobLock", job, mock.Anything).Return(true, nil)
		queue.On("EnqueueToProcess", mock.Anything).Return(errors.New("fail"))
		queue.On("EnqueuePending", mock.Anything).Return(nil)

		manager := worker.NewTaskManager(queue)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager.Start(ctx)

		gomega.RegisterTestingT(t)
		gomega.Eventually(func() bool {
			return queue.AssertCalled(&testing.T{}, "PollPending")
		}).Should(gomega.BeTrue())

		time.Sleep(time.Millisecond * 10)
		queue.AssertCalled(t, "EnqueuePending", job)
	}
}

func TestTaskManagerDoNotSetJobRunningWhenFailToGetAnProcessingJob(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(nil, errors.New("fail"))

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "GetJobByID", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertNotCalled(t, "SetJob", job)
}

func TestTaskManagerDoNotSetJobRunningWhenItIsAlreadyRunning(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task", LastUpdate: time.Now()}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(job, nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "GetJobByID", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertNotCalled(t, "SetJob", job)
}

func TestTaskManagerSetJobRunningWhenItsLastUpdateIsGreaterThanHeartbeatInterval(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task",
		LastUpdate: time.Now().Add(-worker.JobHeartbeatInterval * 2)}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(job, nil)
	queue.On("SetJob", mock.Anything).Return(nil)
	queue.On("EnqueuePending", mock.Anything).Return(nil)
	queue.On("RemoveFromProcessingQueue", mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "GetJobByID", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertCalled(t, "SetJob", job)
}

func TestTaskManagerDoNotProcessWhenTaskWhenItFailsToSetJobRunning(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(nil, nil)
	queue.On("SetJob", job).Return(errors.New("fail"))

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJob", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	gomega.Consistently(func() int {
		return manager.RunningJobs()
	}).Should(gomega.Equal(0))
}

func TestTaskManagerDoNotReenqueueAJobWithAnUnregisteredTaskWhenItFailsToRemoveFromProcessingQueue(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(nil, nil)
	queue.On("SetJob", job).Return(nil)
	queue.On("EnqueuePending", mock.Anything).Return(errors.New("fail"))

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "EnqueuePending", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertNotCalled(t, "RemoveFromProcessingQueue", job)
}

func TestTaskManagerReenqueueAJobWithAnUnregisteredTaskAndRemoveItFromProcessingQueue(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(nil, nil)
	queue.On("SetJob", job).Return(nil)
	queue.On("EnqueuePending", mock.Anything).Return(nil)
	queue.On("RemoveFromProcessingQueue", mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJob", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertCalled(t, "EnqueuePending", job)
	queue.AssertCalled(t, "RemoveFromProcessingQueue", job)
}

func TestTaskManagerRunJobsOnProcessingQueue(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess").Return(job, nil)
	queue.On("GetJobByID", job.ID).Return(nil, nil)
	queue.On("SetJob", job).Return(nil)
	queue.On("EnqueuePending", mock.Anything).Return(nil)
	queue.On("RemoveFromProcessingQueue", mock.Anything).Return(nil)
	queue.On("HeartbeatJob", mock.Anything, mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue)
	task := &mocks.Task{}
	task.On("ID").Return(job.TaskID)
	task.On("Run", mock.Anything, mock.Anything).Return(nil).
		WaitUntil(time.After(time.Millisecond * 100))
	manager.Add(task)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJob", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)

	hostname, _ := os.Hostname()
	queue.AssertCalled(t, "SetJob", mock.MatchedBy(func(j *worker.Job) bool {
		return assert.Equal(t, hostname, job.Host,
			"hostname must be set for running job")
	}))
	queue.AssertNotCalled(t, "EnqueuePending", job)
	gomega.Eventually(manager.RunningJobs).
		Should(gomega.BeNumerically(">=", 1))
}
