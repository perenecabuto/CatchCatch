package worker_test

import (
	"testing"

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

	runChan := make(chan map[string]string)
	worker1.job = func(params map[string]string) error {
		runChan <- params
		return nil
	}

	manager.Start()
	manager.Add(worker1)

	expected := map[string]string{
		"param1": "value1", "param2": "value2",
	}
	err := manager.Run(worker1, expected)
	assert.NoError(t, err)

	actual := <-runChan
	assert.Equal(t, expected, actual)
}

