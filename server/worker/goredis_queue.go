package worker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

// Default keys for job queue control on redis
const (
	DefaultPendingQueue    = "catchcatch:worker:queue"
	DefaultProcessingQueue = "catchcatch:worker:processing"
	DefaultJobsStore       = "catchcatch:worker:jobs"
)

// GoredisTaskQueue is an implemantation of TaskManagerQueue
type GoredisTaskQueue struct {
	redis           redis.Cmdable
	ProcessingQueue string
	PendingQueue    string
	JobsStore       string
}

var _ TaskManagerQueue = (*GoredisTaskQueue)(nil)

// NewGoredisTaskQueue creates a new GoredisTaskQueue
func NewGoredisTaskQueue(r redis.Cmdable) *GoredisTaskQueue {
	return &GoredisTaskQueue{redis: r,
		PendingQueue:    DefaultPendingQueue,
		ProcessingQueue: DefaultProcessingQueue,
		JobsStore:       DefaultJobsStore}
}

// EnqueuePending adds a job to the pending queue
func (q *GoredisTaskQueue) EnqueuePending(job *Job) error {
	encoded, err := json.Marshal(job)
	if err != nil {
		return errors.Cause(err)
	}
	err = q.redis.LPush(q.PendingQueue, encoded).Err()
	return errors.Cause(err)
}

// PollPending pops from pending queue a job
func (q *GoredisTaskQueue) PollPending() (*Job, error) {
	val, err := q.redis.RPop(q.PendingQueue).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Cause(err)
	}
	job, encoded := &Job{}, val
	err = json.Unmarshal([]byte(encoded), job)
	if err != nil {
		return nil, errors.Cause(err)
	}
	return job, nil
}

// EnqueueToProcess set the job on tasks set and add it to the processin queue
func (q *GoredisTaskQueue) EnqueueToProcess(job *Job) error {
	encoded, err := json.Marshal(job)
	if err != nil {
		return errors.Cause(err)
	}
	_, err = q.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		_, err := pipe.HSet(q.JobsStore, job.ID, string(encoded)).Result()
		if err != nil {
			return err
		}
		z := redis.Z{Member: job.ID, Score: 0}
		_, err = q.redis.ZAdd(q.ProcessingQueue, z).Result()
		return err
	})
	return errors.Cause(err)
}

// IsJobOnProcessQueue check is job is in tasks set
func (q *GoredisTaskQueue) IsJobOnProcessQueue(job *Job) (bool, error) {
	exists, err := q.redis.HExists(q.JobsStore, job.ID).Result()
	return exists, errors.Cause(err)
}

// PollProcess get jobs by its last updated time on process queue
func (q *GoredisTaskQueue) PollProcess(updatedUntil time.Time) (*Job, error) {
	zrange := redis.ZRangeBy{Min: "0", Max: strconv.Itoa(int(updatedUntil.UnixNano())), Count: 1}
	vals, err := q.redis.ZRangeByScoreWithScores(q.ProcessingQueue, zrange).Result()
	if len(vals) == 0 || err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Cause(err)
	}
	z := vals[0]
	id, timestamp := z.Member.(string), z.Score
	val, err := q.redis.HGet(q.JobsStore, id).Result()
	if len(vals) == 0 || err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Cause(err)
	}
	job, encoded := &Job{}, val
	err = json.Unmarshal([]byte(encoded), job)
	if err != nil {
		return nil, errors.Cause(err)
	}
	job.LastUpdate = time.Unix(0, int64(timestamp))
	return job, nil
}

// RemoveFromProcessingQueue remove the job from tasks set and remove it from processing queue
func (q *GoredisTaskQueue) RemoveFromProcessingQueue(job *Job) error {
	_, err := q.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		_, err := pipe.HDel(q.JobsStore, job.ID).Result()
		if err != nil {
			return err
		}
		err = pipe.ZRem(q.ProcessingQueue, job.ID).Err()
		return err
	})
	return errors.Cause(err)
}

// SetJobRunning set the job running host and set a lock to avoid race conditions
func (q *GoredisTaskQueue) SetJobRunning(job *Job, host string, lockDuration time.Duration) (bool, error) {
	lock := q.JobLockName(job)
	aquired, err := q.redis.SetNX(lock, host, lockDuration).Result()
	if err != nil {
		return false, errors.Cause(err)
	}
	if !aquired {
		return false, nil
	}
	job.Host = host
	encoded, err := json.Marshal(job)
	if err != nil {
		return false, errors.Cause(err)
	}
	err = q.HeartbeatJob(job)
	if err != nil {
		return false, errors.Cause(err)
	}
	_, err = q.redis.HSet(q.JobsStore, job.ID, string(encoded)).Result()
	if err != nil {
		return false, err
	}
	return true, nil
}

// HeartbeatJob notifies that job is running
func (q *GoredisTaskQueue) HeartbeatJob(job *Job) error {
	timestamp := time.Now().UnixNano()
	z := redis.Z{Member: job.ID, Score: float64(timestamp)}
	_, err := q.redis.ZAdd(q.ProcessingQueue, z).Result()
	return errors.Cause(err)
}

// JobsOnProcessQueue return the jobs set to process
func (q *GoredisTaskQueue) JobsOnProcessQueue() ([]*Job, error) {
	encJobs, err := q.redis.HGetAll(q.JobsStore).Result()
	if err != nil {
		return nil, err
	}
	jobs := make([]*Job, 0)
	for _, encoded := range encJobs {
		job := &Job{}
		json.Unmarshal([]byte(encoded), job)
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Flush clean up all queues and task set
func (q *GoredisTaskQueue) Flush() error {
	err := q.redis.Del(q.JobsStore).Err()
	if err != nil {
		return errors.Cause(err)
	}
	err = q.redis.Del(q.ProcessingQueue).Err()
	if err != nil {
		return errors.Cause(err)
	}
	err = q.redis.Del(q.PendingQueue).Err()
	return errors.Cause(err)
}

// JobLockName generates a lock string for job
func (q *GoredisTaskQueue) JobLockName(job *Job) string {
	return fmt.Sprintf("%s:task:%s", q.ProcessingQueue, job.ID)
}
