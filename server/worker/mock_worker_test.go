package worker_test

import (
	"log"
	"time"
)

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
