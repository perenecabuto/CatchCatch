package worker_test

import (
	"context"
	"errors"
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
	manager := worker.NewTaskManager(queue, "localhost")

	task1 := &mocks.Task{}
	task1.On("ID").Return("task-1")
	task2 := &mocks.Task{}
	task2.On("ID").Return("task-2")

	manager.Add(task1)
	manager.Add(task2)

	actual := manager.TasksID()
	assert.Contains(t, actual, "task-1")
	assert.Contains(t, actual, "task-2")
}

func TestTaskManagerGetTasksByID(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	manager := worker.NewTaskManager(queue, "localhost")

	task1 := &mocks.Task{}
	task1.On("ID").Return("task-1")

	manager.Add(task1)

	actual, err := manager.GetTaskByID("task-1")
	require.NoError(t, err)
	assert.Equal(t, actual, task1)
}

func TestTaskManagerGetTasksByIDReturnAnErrorWhenTaskIsNotRegistered(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	manager := worker.NewTaskManager(queue, "localhost")

	task, err := manager.GetTaskByID("task-1")
	assert.Nil(t, task)
	assert.Error(t, err)
}

func TestTaskManagerStart(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess", mock.Anything).Return(nil, nil)
	manager := worker.NewTaskManager(queue, "localhost")
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
	queue.On("PollProcess", mock.Anything).Return(nil, nil)

	manager := worker.NewTaskManager(queue, "localhost")
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

func TestTaskManagerDoNotAddJobToProcessQueueWhenItFailsToPollPending(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	queue.On("PollPending").Return(nil, errors.New("an error"))
	queue.On("PollProcess", mock.Anything).Return(nil, nil)

	manager := worker.NewTaskManager(queue, "localhost")
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

func TestTaskManagerReenqueueJobWhenItFailsToCheckIfItIsOnProcessQueue(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	job := &worker.Job{
		ID:     "reenqueued",
		TaskID: "task",
	}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess", mock.Anything).Return(nil, nil)
	queue.On("IsJobOnProcessQueue", job).Return(false, errors.New("fail"))
	queue.On("EnqueuePending", mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue, "localhost")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "IsJobOnProcessQueue", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)

	queue.AssertCalled(t, "EnqueuePending", job)
}

func TestTaskManagerDiscardJobWithSpecificIDWhenItIsOnProcessQueue(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	job := &worker.Job{
		ID:     "unique",
		TaskID: "task",
	}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess", mock.Anything).Return(nil, nil)
	queue.On("IsJobOnProcessQueue", job).Return(true, nil)

	manager := worker.NewTaskManager(queue, "localhost")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "IsJobOnProcessQueue", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertNotCalled(t, "EnqueueToProcess", mock.Anything)
}

func TestTaskManagerEnqueueJobsToProcess(t *testing.T) {
	jobs := []*worker.Job{
		{ID: "unique", TaskID: "task"},
		{ID: "ordinary", TaskID: "task"},
	}
	for _, job := range jobs {
		queue := &mocks.TaskManagerQueue{}
		queue.On("PollProcess", mock.Anything).Return(nil, nil)
		queue.On("PollPending").Return(job, nil)
		queue.On("IsJobOnProcessQueue", job).Return(false, nil)
		queue.On("EnqueueToProcess", mock.Anything).Return(nil)

		manager := worker.NewTaskManager(queue, "localhost")
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
		{ID: "unique", TaskID: "task"},
		{ID: "ordinary", TaskID: "task"},
	}
	for _, job := range jobs {
		queue := &mocks.TaskManagerQueue{}
		queue.On("PollProcess", mock.Anything).Return(nil, nil)
		queue.On("PollPending").Return(job, nil)
		queue.On("IsJobOnProcessQueue", job).Return(false, nil)
		queue.On("EnqueueToProcess", mock.Anything).Return(errors.New("fail"))
		queue.On("EnqueuePending", mock.Anything).Return(nil)

		manager := worker.NewTaskManager(queue, "localhost")
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

func TestTaskManagerDoNotProcessJobWhenItFailsToSetJobRunning(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess", mock.Anything).Return(job, nil)
	queue.On("SetJobRunning", job, mock.Anything, mock.Anything).Return(false, errors.New("fail"))

	manager := worker.NewTaskManager(queue, "localhost")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJobRunning", mock.Anything, mock.Anything, mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)

	gomega.Consistently(func() []string {
		return manager.RunningJobs()
	}).Should(gomega.HaveLen(0))
}

func TestTaskManagerDoNotSetJobRunningWhenItIsOnProcessQueue(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess", mock.Anything).Return(job, nil)
	queue.On("SetJobRunning", job, mock.Anything, mock.Anything).Return(false, nil)

	manager := worker.NewTaskManager(queue, "localhost")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJobRunning", mock.Anything, mock.Anything, mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	gomega.Consistently(func() []string {
		return manager.RunningJobs()
	}).Should(gomega.HaveLen(0))
}

func TestTaskManagerDoNotRemoveJobFromProcessingQueueWithAnUnregisteredTaskWhenItFailsToReenqueueIt(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess", mock.Anything).Return(job, nil)
	queue.On("SetJobRunning", job, mock.Anything, mock.Anything).Return(true, nil)
	queue.On("EnqueuePending", mock.Anything).Return(errors.New("fail"))

	manager := worker.NewTaskManager(queue, "localhost")
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
	queue.On("PollProcess", mock.Anything).Return(job, nil)
	queue.On("SetJobRunning", job, mock.Anything, mock.Anything).Return(true, nil)
	queue.On("EnqueuePending", mock.Anything).Return(nil)
	queue.On("RemoveFromProcessingQueue", mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue, "localhost")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() bool {
		return queue.AssertCalled(&testing.T{}, "SetJobRunning", mock.Anything, mock.Anything, mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)
	queue.AssertCalled(t, "EnqueuePending", job)
	queue.AssertCalled(t, "RemoveFromProcessingQueue", job)
}

func TestTaskManagerRunJobsOnProcessingQueue(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}

	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(nil, nil)
	queue.On("PollProcess", mock.Anything).Return(job, nil)
	queue.On("SetJobRunning", job, mock.Anything, mock.Anything).Return(true, nil)
	queue.On("EnqueuePending", mock.Anything).Return(nil)
	queue.On("RemoveFromProcessingQueue", mock.Anything).Return(nil)
	queue.On("HeartbeatJob", mock.Anything, mock.Anything).Return(nil)

	manager := worker.NewTaskManager(queue, "localhost")
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
		return queue.AssertCalled(&testing.T{}, "RemoveFromProcessingQueue", mock.Anything)
	}).Should(gomega.BeTrue())

	time.Sleep(time.Millisecond * 10)

	queue.AssertCalled(t, "RemoveFromProcessingQueue", job)
	queue.AssertNotCalled(t, "EnqueuePending", job)
	gomega.Eventually(manager.RunningJobs).
		Should(gomega.HaveLen(1))
}
