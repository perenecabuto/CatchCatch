package worker_test

import (
	"log"
	"time"
)

type mockWorker struct {
	id  string
	job func(params map[string]string) error
}

func (w *mockWorker) ID() string {
	return w.id
}

func (w mockWorker) Job(params map[string]string) error {
	if w.job != nil {
		return w.job(params)
	}
	return defaultJob(w.ID(), params)
}

func defaultJob(workerID string, params map[string]string) error {
	log.Println("Starting job: ", workerID)
	for i := 0; i < 10; i++ {
		<-time.NewTimer(time.Second).C
		log.Println("Last:" + time.Now().String())
	}
	return nil
}
