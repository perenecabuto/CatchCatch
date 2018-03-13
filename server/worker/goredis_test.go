package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/stretchr/testify/assert"
)

var (
	opts = &redis.Options{Addr: "localhost:6379"}

	worker1    = &mockWorker{id: "worker1"}
	dupWorker1 = &mockWorker{id: "worker1"}
	worker2    = &mockWorker{id: "worker2"}
	dupWorker2 = &mockWorker{id: "worker1"}
	worker3    = &mockWorker{id: "worker3"}
	dupWorker3 = &mockWorker{id: "worker1"}
)

func TestGoredisWorkerManagerAddWorker(t *testing.T) {
	client := redis.NewClient(opts)
	manager := worker.NewGoredisWorkerManager(client)

	manager.Add(worker1)
	manager.Add(dupWorker1)

	manager.Add(worker2)
	manager.Add(dupWorker2)

	manager.Add(worker3)
	manager.Add(dupWorker3)

	grManager := manager.(*worker.GoredisWorkerManager)

	actualWorkers := grManager.WorkersIDs()

	assert.Len(t, actualWorkers, 3)
	assert.Contains(t, actualWorkers, worker1.ID())
	assert.Contains(t, actualWorkers, worker2.ID())
	assert.Contains(t, actualWorkers, worker3.ID())
	assert.NotContains(t, actualWorkers, "worker4")
}

func TestGoredisWorkerManagerRunItsWorkerTasks(t *testing.T) {
	client := redis.NewClient(opts)
	manager := worker.NewGoredisWorkerManager(client)
	manager.Flush()

	runChan := make(chan map[string]string)
	worker := &mockWorker{id: "worker1", job: func(params map[string]string) error {
		runChan <- params
		return nil
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)
	defer manager.Stop()

	manager.Add(worker)

	expected := map[string]string{
		"param1": "value1", "param2": "value2",
	}
	err := manager.Run(worker, expected)
	assert.NoError(t, err)

	actual := <-runChan
	assert.Equal(t, expected, actual)
}

func TestGoredisWorkerManagerStopWhenContextDone(t *testing.T) {
	client := redis.NewClient(opts)
	manager := worker.NewGoredisWorkerManager(client)

	ctx, cancel := context.WithCancel(context.Background())

	manager.Start(ctx)
	time.Sleep(time.Millisecond * 100)
	assert.True(t, manager.Started())

	cancel()
	time.Sleep(time.Millisecond * 100)
	assert.False(t, manager.Started())
}

func TestGoredisWorkerManagerRunTasks(t *testing.T) {
	client1 := redis.NewClient(opts)
	client2 := redis.NewClient(opts)
	client3 := redis.NewClient(opts)

	manager1 := worker.NewGoredisWorkerManager(client1)
	manager2 := worker.NewGoredisWorkerManager(client2)
	manager3 := worker.NewGoredisWorkerManager(client3)

	manager1.Add(worker1)
	manager1.Add(worker2)
	manager1.Add(worker3)

	manager2.Add(worker1)
	manager2.Add(worker2)
	manager2.Add(worker3)

	manager3.Add(worker1)
	manager3.Add(worker2)
	manager3.Add(worker3)

	manager1.Run(worker1, nil)
	manager1.Run(worker2, nil)
	manager1.Run(worker3, nil)
	manager2.Run(worker1, nil)
	manager2.Run(worker2, nil)
	manager2.Run(worker3, nil)
	manager3.Run(worker1, nil)
	manager3.Run(worker2, nil)
	manager3.Run(worker3, nil)

	runningTasks, err := manager1.BusyWorkers()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(runningTasks))

	ctx := context.Background()
	manager1.Start(ctx)
	manager2.Start(ctx)
	manager3.Start(ctx)

	time.Sleep(time.Second)

	runningTasks, err = manager1.BusyWorkers()
	assert.NoError(t, err)
	assert.Equal(t, 9, len(runningTasks))

	manager1.Stop()
	manager2.Stop()
	manager3.Stop()

	runningTasks, err = manager1.BusyWorkers()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(runningTasks))
}

func TestGoredisWorkerManagerRunUniqueTasks(t *testing.T) {
	client1 := redis.NewClient(opts)
	client2 := redis.NewClient(opts)

	manager1 := worker.NewGoredisWorkerManager(client1)
	manager2 := worker.NewGoredisWorkerManager(client2)

	ctx := context.Background()
	manager1.Start(ctx)
	manager2.Start(ctx)

	manager1.Add(worker1)
	manager1.Add(worker1)
	manager2.Add(worker1)
	manager2.Add(worker1)

	manager1.RunUnique(worker1, nil)
	manager1.RunUnique(worker1, nil)
	manager2.RunUnique(worker1, nil)
	manager2.RunUnique(worker1, nil)

	time.Sleep(time.Millisecond * 100)

	runningTasks, err := manager1.BusyWorkers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(runningTasks))

	manager1.Stop()
	manager2.Stop()

	runningTasks, err = manager1.BusyWorkers()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(runningTasks))
}
