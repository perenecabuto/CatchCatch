package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/stretchr/testify/suite"
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

type GoRedisSuite struct {
	suite.Suite
	client *redis.Client
}

func TestGoRedis(t *testing.T) {
	suite.Run(t, &GoRedisSuite{})
}

func (t *GoRedisSuite) SetupTest() {
	t.client = redis.NewClient(opts)
	err := t.client.Ping().Err()
	if err != nil {
		t.T().Skip(err)
		return
	}

	t.client.FlushAll()
}

func (s *GoRedisSuite) TestGoredisWorkerManagerAddWorker() {
	manager := worker.NewGoredisWorkerManager(s.client)

	manager.Add(worker1)
	manager.Add(dupWorker1)

	manager.Add(worker2)
	manager.Add(dupWorker2)

	manager.Add(worker3)
	manager.Add(dupWorker3)

	grManager := manager.(*worker.GoredisWorkerManager)

	actualWorkers := grManager.WorkersIDs()

	s.Assert().Len(actualWorkers, 3)
	s.Assert().Contains(actualWorkers, worker1.ID())
	s.Assert().Contains(actualWorkers, worker2.ID())
	s.Assert().Contains(actualWorkers, worker3.ID())
	s.Assert().NotContains(actualWorkers, "worker4")
}

func (s *GoRedisSuite) TestGoredisWorkerManagerRunItsWorkerTasks() {
	manager := worker.NewGoredisWorkerManager(s.client)
	manager.Flush()

	runChan := make(chan worker.TaskParams)
	w := &mockWorker{id: "worker1", run: func(params worker.TaskParams) error {
		runChan <- params
		return nil
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.Start(ctx)
	defer manager.Stop()

	manager.Add(w)

	expected := worker.TaskParams{
		"param1": "value1", "param2": "value2",
	}
	err := manager.Run(w, expected)
	s.Require().NoError(err)

	actual := <-runChan
	s.Assert().Equal(expected, actual)
}

func (s *GoRedisSuite) TestGoredisWorkerManagerStopWhenContextDone() {
	manager := worker.NewGoredisWorkerManager(s.client)

	ctx, cancel := context.WithCancel(context.Background())

	manager.Start(ctx)
	time.Sleep(time.Millisecond * 100)
	s.Assert().True(manager.Started())

	cancel()
	time.Sleep(time.Millisecond * 100)
	s.Assert().False(manager.Started())
}

func (s *GoRedisSuite) TestGoredisWorkerManagerRunTasks() {
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

	runningTasks, err := manager1.RunningTasks()
	s.Require().NoError(err)
	s.Assert().Equal(0, len(runningTasks))

	ctx := context.Background()
	manager1.Start(ctx)
	manager2.Start(ctx)
	manager3.Start(ctx)

	time.Sleep(time.Second)

	runningTasks, err = manager1.RunningTasks()
	s.Require().NoError(err)
	s.Assert().Equal(9, len(runningTasks))

	manager1.Stop()
	manager2.Stop()
	manager3.Stop()

	runningTasks, err = manager1.RunningTasks()
	s.Require().NoError(err)
	s.Assert().Equal(0, len(runningTasks))
}

func (s *GoRedisSuite) TestGoredisWorkerManagerRunUniqueTasks() {
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

	runningTasks, err := manager1.RunningTasks()
	s.Require().NoError(err)
	s.Assert().Equal(1, len(runningTasks))

	manager1.Stop()
	manager2.Stop()

	runningTasks, err = manager1.RunningTasks()
	s.Require().NoError(err)
	s.Assert().Equal(0, len(runningTasks))
}
