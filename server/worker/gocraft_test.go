package worker_test

import (
	"log"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/gocraft/work"
	"github.com/stretchr/testify/assert"

	"github.com/perenecabuto/CatchCatch/server/worker"
)

func TestMain(m *testing.M) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	go func() {
		for range signalChan {
			log.Println("Received an interrupt, stopping services...")
			os.Exit(0)
		}
	}()

	os.Exit(m.Run())
}

func TestGocraftWorkerManager(t *testing.T) {
	_, err := redis.Dial("tcp", "localhost:6379")
	if err != nil {
		t.Skip("Redis connection error:", err)
		return
	}

	redisPool1 := &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}
	redisPool2 := &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}
	redisPool3 := &redis.Pool{
		MaxActive: 5,
		MaxIdle:   5,
		Wait:      true,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", "localhost:6379")
		},
	}

	manager1 := worker.NewGocraftWorkerManager(redisPool1)
	manager2 := worker.NewGocraftWorkerManager(redisPool2)
	manager3 := worker.NewGocraftWorkerManager(redisPool3)

	manager1.Start()
	manager2.Start()
	manager3.Start()

	worker := mockWorker("worker1")

	manager1.Add(worker)
	manager2.Add(worker)
	manager3.Add(worker)

	err = manager1.Run(worker, nil)
	assert.NoError(t, err)
	err = manager1.Run(worker, nil)
	assert.NoError(t, err)

	err = manager2.Run(worker, nil)
	assert.NoError(t, err)

	err = manager2.Run(worker, nil)
	assert.NoError(t, err)

	err = manager3.Run(worker, nil)
	assert.NoError(t, err)

	err = manager3.Run(worker, nil)
	assert.NoError(t, err)

	time.Sleep(time.Second * 10)

	client := work.NewClient("catchcatch", redisPool1)
	observations, _ := client.WorkerObservations()
	var busyObservations []*work.WorkerObservation
	for _, ob := range observations {
		if ob.IsBusy {
			busyObservations = append(busyObservations, ob)
		}
	}

	go manager1.Stop()
	go manager2.Stop()
	go manager3.Stop()

	time.Sleep(time.Second * 10)

	assert.Equal(t, 1, len(busyObservations))
}

type mockWorker string

func (w mockWorker) ID() string {
	return string(w)
}

func (w mockWorker) Job(params map[string]interface{}) error {
	log.Println("Starting job: ", w.ID())
	for i := 0; i < 10; i++ {
		<-time.NewTimer(time.Second).C
		log.Println("Last:" + time.Now().String())
	}
	return nil
}
