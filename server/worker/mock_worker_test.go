package worker_test

import (
	"context"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/worker"
)

type mockWorker struct {
	id  string
	run func(params worker.TaskParams) error
}

func (w *mockWorker) ID() string {
	return w.id
}

func (w mockWorker) Run(params map[string]interface{}) error {
	if w.run != nil {
		return w.run(params)
	}
	return defaultJob(w.ID(), params)
}

func defaultJob(workerID string, params worker.TaskParams) error {
	log.Println("Running worker: ", workerID)
	for i := 0; i < 10; i++ {
		<-time.NewTimer(time.Millisecond * 100).C
	}
	return nil
}
