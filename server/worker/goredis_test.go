package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"

	"github.com/perenecabuto/CatchCatch/server/worker"
)

var (
	opts = &redis.Options{Addr: "redis:6379"}

	worker1  = &mockTask{id: "worker1"}
	dupTask1 = &mockTask{id: "worker1"}
	worker2  = &mockTask{id: "worker2"}
	dupTask2 = &mockTask{id: "worker1"}
	worker3  = &mockTask{id: "worker3"}
	dupTask3 = &mockTask{id: "worker1"}
)

type GoRedisSuite struct {
	suite.Suite
	client redis.Cmdable
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

	worker.QueuePollInterval = time.Millisecond
}

func (s *GoRedisSuite) TestGoredisTaskManagerAddTask() {
	manager := worker.NewGoredisTaskManager(s.client, "server-1")

	manager.Add(worker1)
	manager.Add(dupTask1)

	manager.Add(worker2)
	manager.Add(dupTask2)

	manager.Add(worker3)
	manager.Add(dupTask3)

	grManager := manager.(*worker.GoredisTaskManager)

	actualTasks := grManager.TasksID()

	s.Assert().Len(actualTasks, 3)
	s.Assert().Contains(actualTasks, worker1.ID())
	s.Assert().Contains(actualTasks, worker2.ID())
	s.Assert().Contains(actualTasks, worker3.ID())
	s.Assert().NotContains(actualTasks, "worker4")
}

func (s *GoRedisSuite) TestGoredisTaskManagerRunItsTaskJobs() {
	manager := worker.NewGoredisTaskManager(s.client, "server-1")
	manager.Flush()

	runChan := make(chan worker.TaskParams)
	w := &mockTask{id: "worker1", run: func(params worker.TaskParams) error {
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

func (s *GoRedisSuite) TestGoredisTaskManagerStopWhenContextDone() {
	ctx, cancel := context.WithCancel(context.Background())

	manager := worker.NewGoredisTaskManager(s.client, "server-1")
	manager.Start(ctx)
	time.Sleep(time.Millisecond * 100)
	s.Assert().True(manager.Started())

	cancel()
	time.Sleep(time.Millisecond * 100)
	s.Assert().False(manager.Started())
}

func (s *GoRedisSuite) TestGoredisTaskManagerRunJobs() {
	client1 := redis.NewClient(opts)
	client2 := redis.NewClient(opts)
	client3 := redis.NewClient(opts)

	manager1 := worker.NewGoredisTaskManager(client1, "server-1")
	manager2 := worker.NewGoredisTaskManager(client2, "server-2")
	manager3 := worker.NewGoredisTaskManager(client3, "server-3")

	manager1.Add(worker1)
	manager1.Add(worker2)
	manager1.Add(worker3)

	manager2.Add(worker1)
	manager2.Add(worker2)
	manager2.Add(worker3)

	manager3.Add(worker1)
	manager3.Add(worker2)
	manager3.Add(worker3)

	runningJobs, err := manager1.ProcessingJobs()
	s.Require().NoError(err)
	s.Assert().Equal(0, len(runningJobs))

	manager1.Run(worker1, nil)
	manager1.Run(worker2, nil)
	manager1.Run(worker3, nil)
	manager2.Run(worker1, nil)
	manager2.Run(worker2, nil)
	manager2.Run(worker3, nil)
	manager3.Run(worker1, nil)
	manager3.Run(worker2, nil)
	manager3.Run(worker3, nil)

	ctx := context.Background()
	manager1.Start(ctx)
	manager2.Start(ctx)
	manager3.Start(ctx)

	gomega.RegisterTestingT(s.T())
	gomega.Eventually(manager1.ProcessingJobs, time.Second).Should(gomega.HaveLen(9))

	manager1.Stop()
	manager2.Stop()
	manager3.Stop()

	gomega.Eventually(manager1.ProcessingJobs, time.Second).Should(gomega.HaveLen(0))
}

func (s *GoRedisSuite) TestGoredisTaskManagerRunUniqueJobs() {
	client1 := redis.NewClient(opts)
	client2 := redis.NewClient(opts)

	manager1 := worker.NewGoredisTaskManager(client1, "server-1")
	manager2 := worker.NewGoredisTaskManager(client2, "server-2")

	ctx := context.Background()
	manager1.Start(ctx)
	manager2.Start(ctx)

	manager1.Add(worker1)
	manager1.Add(worker1)
	manager2.Add(worker1)
	manager2.Add(worker1)

	manager1.RunUnique(worker1, nil, "unique-1")
	manager1.RunUnique(worker1, nil, "unique-2")
	manager2.RunUnique(worker1, nil, "unique-3")
	manager2.RunUnique(worker1, nil, "unique-4")

	time.Sleep(time.Millisecond * 100)

	processingJobs, err := manager1.ProcessingJobs()
	s.Require().NoError(err)
	s.Assert().Equal(1, len(processingJobs))

	manager1.Stop()
	manager2.Stop()

	processingJobs, err = manager1.ProcessingJobs()
	s.Require().NoError(err)
	s.Assert().Equal(0, len(processingJobs))
}
