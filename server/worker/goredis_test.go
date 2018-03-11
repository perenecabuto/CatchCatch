package worker_test

import (
	"testing"

	"github.com/go-redis/redis"
	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/stretchr/testify/assert"
)

var opts = &redis.Options{Addr: "localhost:6379"}

func TestGoredisWorkerManagerAddWorker(t *testing.T) {
	client := redis.NewClient(opts)
	manager := worker.NewGoredisWorkerManager(client)

	worker1 := mockWorker("worker1")
	dupWorker1 := mockWorker("worker1")
	worker2 := mockWorker("worker2")
	dupWorker2 := mockWorker("worker1")
	worker3 := mockWorker("worker3")
	dupWorker3 := mockWorker("worker1")
	worker4 := mockWorker("worker4")

	manager.Add(worker1)
	manager.Add(dupWorker1)

	manager.Add(worker2)
	manager.Add(dupWorker2)

	manager.Add(worker3)
	manager.Add(dupWorker3)

	grManager := manager.(*worker.GoredisWorkerManager)

	actualWorkers := grManager.Workers()

	assert.Len(t, actualWorkers, 3)
	assert.Contains(t, actualWorkers, worker1)
	assert.Contains(t, actualWorkers, worker2)
	assert.Contains(t, actualWorkers, worker3)
	assert.NotContains(t, actualWorkers, worker4)
}
