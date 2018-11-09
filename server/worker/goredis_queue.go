package worker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"
)

const (
	pendingQueue    = "catchcatch:worker:queue"
	processingQueue = "catchcatch:worker:processing"
	tasksSet        = "catchcatch:worker:tasks"
)

type GoredisTaskQueue struct {
	redis           redis.Cmdable
	processingQueue string
	pendingQueue    string
	tasksSet        string
}

var _ TaskManagerQueue = (*GoredisTaskQueue)(nil)

func NewGoredisTaskQueue(r redis.Cmdable) *GoredisTaskQueue {
	return &GoredisTaskQueue{redis: r,
		pendingQueue: pendingQueue, processingQueue: processingQueue, tasksSet: tasksSet}
}

func (q *GoredisTaskQueue) IsJobOnProcessQueue(job *Job) (bool, error) {
	exists, err := q.redis.HExists(q.tasksSet, job.ID).Result()
	return exists, errors.Cause(err)
}

func (q *GoredisTaskQueue) PollPending() (*Job, error) {
	val, err := q.redis.RPop(q.pendingQueue).Result()
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

func (q *GoredisTaskQueue) PollProcess(updatedOn time.Time) (*Job, error) {
	zrange := redis.ZRangeBy{Min: "0", Max: strconv.Itoa(int(updatedOn.Unix())), Count: 1}
	vals, err := q.redis.ZRangeByScoreWithScores(q.processingQueue, zrange).Result()
	if len(vals) == 0 || err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Cause(err)
	}
	z := vals[0]
	id, timestamp := z.Member.(string), z.Score
	val, err := q.redis.HGet(q.tasksSet, id).Result()
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
	job.LastUpdate = time.Unix(int64(timestamp), 0)
	return job, nil
}

func (q *GoredisTaskQueue) EnqueuePending(job *Job) error {
	encoded, err := json.Marshal(job)
	if err != nil {
		return errors.Cause(err)
	}
	err = q.redis.LPush(q.pendingQueue, encoded).Err()
	return errors.Cause(err)
}

func (q *GoredisTaskQueue) EnqueueToProcess(job *Job) error {
	encoded, err := json.Marshal(job)
	if err != nil {
		return errors.Cause(err)
	}
	_, err = q.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		_, err := pipe.HSet(q.tasksSet, job.ID, string(encoded)).Result()
		if err != nil {
			return err
		}
		z := redis.Z{Member: job.ID, Score: 0}
		_, err = q.redis.ZAdd(q.processingQueue, z).Result()
		return err
	})
	return errors.Cause(err)
}

func (q *GoredisTaskQueue) RemoveFromProcessingQueue(job *Job) error {
	_, err := q.redis.TxPipelined(func(pipe redis.Pipeliner) error {
		_, err := pipe.HDel(q.tasksSet, job.ID).Result()
		if err != nil {
			return err
		}
		err = pipe.ZRem(q.processingQueue, job.ID).Err()
		return err
	})
	return errors.Cause(err)
}

func (q *GoredisTaskQueue) SetJobRunning(job *Job, host string, lockDuration time.Duration) (bool, error) {
	aquired, err := q.redis.SetNX(fmt.Sprintf("%s:task:%s", q.processingQueue, job.ID), host, lockDuration).Result()
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
	_, err = q.redis.HSet(q.tasksSet, job.ID, string(encoded)).Result()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (q *GoredisTaskQueue) HeartbeatJob(job *Job) error {
	timestamp := time.Now().Unix()
	z := redis.Z{Member: job.ID, Score: float64(timestamp)}
	_, err := q.redis.ZAdd(q.processingQueue, z).Result()
	return errors.Cause(err)
}

func (q *GoredisTaskQueue) JobsOnProcessQueue() ([]*Job, error) {
	encJobs, err := q.redis.HGetAll(q.processingQueue).Result()
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

func (q *GoredisTaskQueue) Flush() error {
	err := q.redis.Del(q.pendingQueue).Err()
	if err != nil {
		return errors.Cause(err)
	}
	err = q.redis.Del(q.processingQueue).Err()
	return errors.Cause(err)
}
