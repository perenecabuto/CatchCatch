package worker_test

import (
	"log"
	"time"
)

type mockWorker struct {
	id  string
	run func(params map[string]interface{}) error
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

func defaultJob(workerID string, params map[string]interface{}) error {
	log.Println("Running worker: ", workerID)
	for i := 0; i < 10; i++ {
		<-time.NewTimer(time.Millisecond * 100).C
	}
	return nil
}
