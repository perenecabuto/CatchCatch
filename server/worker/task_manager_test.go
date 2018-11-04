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
	"github.com/thoas/go-funk"

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

func TestTaskManagerDoNotAddJobToProcessWhenItFailsToPoll(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	queue.On("PollPending").Return(nil, errors.New("an error"))
	queue.On("PollProcess").Return(nil, nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() []string {
		return funk.Map(queue.Calls, func(c mock.Call) string { return c.Method }).([]string)
	}).Should(gomega.ContainElement("PollPending"))

	time.Sleep(time.Millisecond * 10)

	queue.AssertNotCalled(t, "EnqueueToProcess", mock.Anything)
}

func TestTaskManagerReenqueueUniqueJobWhenItFailsToCheckIfItIsProcessing(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	job := &worker.Job{
		ID:         "reenqueued",
		Unique:     true,
		TaskID:     "task",
		LastUpdate: time.Now(),
	}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess").Return(nil, nil)
	queue.On("EnqueuePending", mock.Anything).Return(nil)
	queue.On("GetProcessingJobByTaskAndParams", job.TaskID, job.Params).Return(nil, errors.New("fail"))

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() []string {
		return funk.Map(queue.Calls, func(c mock.Call) string { return c.Method }).([]string)
	}).Should(gomega.ContainElement("GetProcessingJobByTaskAndParams"))

	time.Sleep(time.Millisecond * 10)

	queue.AssertCalled(t, "EnqueuePending", job)
}

func TestTaskManagerDiscardUniqueJobAlreadyRunning(t *testing.T) {
	queue := &mocks.TaskManagerQueue{}

	job := &worker.Job{
		ID:         "unique",
		TaskID:     "task",
		Unique:     true,
		LastUpdate: time.Now(),
	}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess").Return(nil, nil)
	queue.On("GetProcessingJobByTaskAndParams", job.TaskID, job.Params).Return(job, nil)

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() []string {
		return funk.Map(queue.Calls, func(c mock.Call) string { return c.Method }).([]string)
	}).Should(gomega.ContainElement("GetProcessingJobByTaskAndParams"))

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
		queue.On("GetProcessingJobByTaskAndParams", mock.Anything, mock.Anything).Return(nil, nil)
		queue.On("EnqueueToProcess", mock.Anything).Return(nil)

		manager := worker.NewTaskManager(queue)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager.Start(ctx)

		gomega.RegisterTestingT(t)
		gomega.Eventually(func() []string {
			return funk.Map(queue.Calls, func(c mock.Call) string { return c.Method }).([]string)
		}).Should(gomega.ContainElement("PollPending"))

		time.Sleep(time.Millisecond * 10)
		queue.AssertCalled(t, "EnqueueToProcess", job)
	}
}

func TestTaskManagerReenqueueToPendingWhenFailToEnqueueToProcess(t *testing.T) {
	job := &worker.Job{ID: "job", TaskID: "task"}
	queue := &mocks.TaskManagerQueue{}
	queue.On("PollPending").Return(job, nil)
	queue.On("PollProcess").Return(nil, nil)
	queue.On("GetProcessingJobByTaskAndParams", mock.Anything, mock.Anything).Return(nil, nil)
	queue.On("EnqueueToProcess", mock.Anything).Return(errors.New("fail"))
	queue.On("EnqueuePending", mock.Anything).Return(errors.New("fail"))

	manager := worker.NewTaskManager(queue)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Start(ctx)

	gomega.RegisterTestingT(t)
	gomega.Eventually(func() []string {
		return funk.Map(queue.Calls, func(c mock.Call) string { return c.Method }).([]string)
	}).Should(gomega.ContainElement("EnqueueToProcess"))

	time.Sleep(time.Millisecond * 10)
	queue.AssertCalled(t, "EnqueuePending", job)
}
