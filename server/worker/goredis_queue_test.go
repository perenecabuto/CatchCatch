package worker_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/go-redis/redis"
	"github.com/perenecabuto/CatchCatch/server/worker"
	"github.com/stretchr/testify/suite"
)

type GoRedisManagerSuite struct {
	suite.Suite
	client redis.Cmdable
	queue  *worker.GoredisTaskQueue
}

func TestGoRedisManager(t *testing.T) {
	suite.Run(t, &GoRedisManagerSuite{})
}

func (s *GoRedisManagerSuite) SetupTest() {
	client := redis.NewClient(opts)
	if err := client.Ping().Err(); err != nil {
		s.T().Skip(err)
		return
	}
	client.FlushAll()

	s.client = client
	s.queue = worker.NewGoredisTaskQueue(client)
}

func (s *GoRedisManagerSuite) TestEnqueuePendingJobs() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": 2.0}, TaskID: "task-2"}
	err := s.queue.EnqueuePending(job)
	s.Require().NoError(err)

	stored, err := s.client.RPop(s.queue.PendingQueue).Result()
	s.Require().NoError(err)

	actual := &worker.Job{}
	err = json.Unmarshal([]byte(stored), actual)
	s.Require().NoError(err)

	s.Assert().EqualValues(job, actual)
}

func (s *GoRedisManagerSuite) TestNotEnqueuePendingJobsWhenCantEncodeJob() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": make(chan int)}}
	err := s.queue.EnqueuePending(job)
	s.Require().Error(err)

	size, err := s.client.LLen(s.queue.PendingQueue).Result()
	s.Require().NoError(err)

	s.Assert().EqualValues(0, size)
}

func (s *GoRedisManagerSuite) TestPollPending() {
	job1 := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": 2.0}, TaskID: "task-2"}
	job2 := &worker.Job{ID: "456", Params: worker.TaskParams{"k1": "v1", "k2": 2.0}, TaskID: "task-2"}
	err := s.queue.EnqueuePending(job1)
	s.Require().NoError(err)
	err = s.queue.EnqueuePending(job2)
	s.Require().NoError(err)

	got1, err := s.queue.PollPending()
	s.Require().NoError(err)
	got2, err := s.queue.PollPending()
	s.Require().NoError(err)
	got3, err := s.queue.PollPending()
	s.Require().NoError(err)

	s.Assert().EqualValues(job1, got1)
	s.Assert().EqualValues(job2, got2)
	s.Assert().Nil(got3)

}

func (s *GoRedisManagerSuite) TestEnqueueToProcess() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": 2.0}, TaskID: "task-2"}
	err := s.queue.EnqueueToProcess(job)
	s.Require().NoError(err)

	scores, err := s.client.ZRangeWithScores(s.queue.ProcessingQueue, 0, -1).Result()
	s.Require().NoError(err)

	stored, err := s.client.HGet(s.queue.JobsStore, job.ID).Result()
	s.Require().NoError(err)

	actual := &worker.Job{}
	err = json.Unmarshal([]byte(stored), actual)
	s.Require().NoError(err)

	s.Assert().Len(scores, 1)
	s.Assert().EqualValues(job.ID, scores[0].Member)
	s.Assert().EqualValues(0, scores[0].Score)
	s.Assert().EqualValues(job, actual)
}

func (s *GoRedisManagerSuite) TestNotEnqueueJobToProcessWhenCantEncodeJob() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": make(chan int)}}
	err := s.queue.EnqueueToProcess(job)
	s.Require().Error(err)

	scores, err := s.client.ZRangeWithScores(s.queue.ProcessingQueue, 0, -1).Result()
	s.Require().NoError(err)

	s.Assert().Len(scores, 0)
}

func (s *GoRedisManagerSuite) TestRemoveFromProcessingQueue() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": "v2"}}
	s.queue.EnqueueToProcess(job)

	scores := s.client.ZRangeWithScores(s.queue.ProcessingQueue, 0, -1).Val()
	s.Require().Len(scores, 1)
	size := s.client.HLen(s.queue.JobsStore).Val()
	s.Require().EqualValues(1, size)

	err := s.queue.RemoveFromProcessingQueue(job)
	s.Require().NoError(err)

	actualScores := s.client.ZRangeWithScores(s.queue.ProcessingQueue, 0, -1).Val()
	actualJobsStoreSize := s.client.HLen(s.queue.JobsStore).Val()

	s.Assert().Len(actualScores, 0)
	s.Assert().EqualValues(0, actualJobsStoreSize)
}

func (s *GoRedisManagerSuite) TestSetJobRunning() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": "v2"}}
	s.queue.EnqueueToProcess(job)

	host := "test-host"
	aquired, err := s.queue.SetJobRunning(job, host, time.Hour)
	s.Require().NoError(err)

	got := &worker.Job{}
	encoded := s.client.HGet(s.queue.JobsStore, job.ID).Val()
	json.Unmarshal([]byte(encoded), got)

	lock := s.queue.JobLockName(job)

	lockCreated := s.client.Exists(lock).Val() > 0
	lockTTL := s.client.TTL(lock).Val()
	score := s.client.ZScore(s.queue.ProcessingQueue, job.ID).Val()

	s.Assert().True(aquired)
	s.Assert().True(lockCreated)
	s.Assert().EqualValues(60, lockTTL.Minutes())
	s.Assert().Equal(host, got.Host)

	diff := time.Now().Unix() - time.Unix(0, int64(score)).Unix()
	s.Assert().True(diff < 10)
}

func (s *GoRedisManagerSuite) TestNotSetJobRunningWhenAJobWithSameIDIsAlredySet() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": "v2"}}
	err := s.queue.EnqueueToProcess(job)
	s.Require().NoError(err)

	host := "test-host"
	aquired, err := s.queue.SetJobRunning(job, host, time.Hour)
	s.Require().NoError(err)
	s.Require().True(aquired)

	score := s.client.ZScore(s.queue.ProcessingQueue, job.ID).Val()

	for i := 0; i < 5; i++ {
		aquired, err = s.queue.SetJobRunning(job, host, time.Hour)
		s.Require().NoError(err)
		s.Assert().False(aquired)
		retrieved := s.client.ZScore(s.queue.ProcessingQueue, job.ID).Val()
		s.Assert().Equal(score, retrieved)
	}
}

func (s *GoRedisManagerSuite) TestHeartbeatJob() {
	job := &worker.Job{ID: "123", Params: worker.TaskParams{"k1": "v1", "k2": "v2"}}
	err := s.queue.EnqueueToProcess(job)
	s.Require().NoError(err)

	host := "test-host"
	aquired, err := s.queue.SetJobRunning(job, host, time.Hour)
	s.Require().NoError(err)
	s.Require().True(aquired)

	score := s.client.ZScore(s.queue.ProcessingQueue, job.ID).Val()

	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond)
		err = s.queue.HeartbeatJob(job)
		s.Require().NoError(err)
		retrieved := s.client.ZScore(s.queue.ProcessingQueue, job.ID).Val()
		s.Assert().True(retrieved > score)

		score = retrieved
	}
}

func (s *GoRedisManagerSuite) TestIsJobOnProcessQueue() {

}

func (s *GoRedisManagerSuite) TestPollProcess() {

}

func (s *GoRedisManagerSuite) TestJobsOnProcessQueue() {

}
func (s *GoRedisManagerSuite) TestFlush() {

}
