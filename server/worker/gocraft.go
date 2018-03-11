package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/garyburd/redigo/redis"
	"github.com/gocraft/work"
)

// GocraftWorkerManager gocraft/work implementation of worker.Manager
type GocraftWorkerManager struct {
	pool     *work.WorkerPool
	enqueuer *work.Enqueuer
}

// NewGocraftWorkerManager creates a new GocraftWorkerManager
func NewGocraftWorkerManager(pool *redis.Pool) Manager {
	ctx := struct{}{}
	return &GocraftWorkerManager{
		enqueuer: work.NewEnqueuer("catchcatch", pool),
		pool:     work.NewWorkerPool(ctx, 1, "catchcatch", pool),
	}
}

// Start the worker manager pool
func (wm *GocraftWorkerManager) Start() {
	wm.pool.Start()
}

// Stop the worker manager pool
func (wm *GocraftWorkerManager) Stop() {
	wm.pool.Stop()
}

// Add workers to pool
func (wm *GocraftWorkerManager) Add(w Worker) {
	wm.pool.Job(w.ID(), func(job *work.Job) error {
		lockKey := "lock:" + job.Name
		locked, err := writeLock(wm.enqueuer.Pool, lockKey, job.ID, 1000)
		if locked {
			log.Printf("Job: %s already running", job.Name)
			return nil
		}
		if err != nil {
			log.Printf("Job: %s error on %s", job.Name, err.Error())
			return err
		}
		defer releaseLock(wm.enqueuer.Pool, lockKey, job.ID)

		params := make(map[string]string)
		for k, v := range job.Args {
			params[k] = fmt.Sprint(v)
		}
		return w.Job(params)
	})
}

// Run worker to process task
func (wm *GocraftWorkerManager) Run(w Worker, params map[string]string) error {
	args := make(map[string]interface{})
	for k, v := range params {
		args[k] = v
	}
	_, err := wm.enqueuer.Enqueue(w.ID(), args)
	return err
}

// writeLock attempts to grab a redis lock. The error returned is safe to ignore
// if all you care about is whether or not the lock was acquired successfully.
func writeLock(pool *redis.Pool, name, secret string, ttl uint64) (bool, error) {
	rc := pool.Get()
	defer rc.Close()
	resp, err := rc.Do("SET", name, secret, "PX", ttl, "NX")
	if err != nil {
		return false, err
	}
	return resp == nil, nil
}

// writeLock releases the redis lock
func releaseLock(pool *redis.Pool, name, secret string) (bool, error) {
	rc := pool.Get()
	defer rc.Close()
	script := redis.NewScript(1, unlockScript)
	resp, err := redis.Int(script.Do(rc, name, secret))
	if err != nil {
		return false, err
	}
	return resp == 0, nil
}

const lockScript = `
local v = redis.call("GET", KEYS[1])
if v == false or v == ARGV[1]
then
	return redis.call("SET", KEYS[1], ARGV[1], "EX", ARGV[2]) and 1
else
	return 0
end
`

const unlockScript = `
local v = redis.call("GET",KEYS[1])
if v == false then
	return 1
elseif v == ARGV[1] then
	return redis.call("DEL",KEYS[1])
else
	return 0
end
`
