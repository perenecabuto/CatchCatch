package worker_test

import (
	"log"
	"os"
	"os/signal"
	"testing"

	"github.com/go-redis/redis"
)

func TestMain(m *testing.M) {
	client := redis.NewClient(opts)
	err := client.Ping().Err()
	if err != nil {
		log.Println("Skip redis tests:", err)
		return
	}

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
