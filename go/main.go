package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"

	io "github.com/googollee/go-socket.io"
	redis "gopkg.in/redis.v5"
)

var (
	redisAddress   = flag.String("redis-addr", "localhost:9851", "redis address")
	maxConnections = flag.Int("redis-connections", 100, "redis address")
)

func main() {
	pool := mustRedisConnect()
	service := &PlayerLocationService{pool}
	server, err := io.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("error", func(so io.Socket, err error) {
		log.Println("error:", err)
	})
	eventH := NewEventHandler(server, service)

	http.Handle("/socket.io/", eventH)
	http.Handle("/", http.FileServer(http.Dir("../web")))
	log.Println("Serving at localhost: 5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}

func mustRedisConnect() *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: *redisAddress, PoolSize: 1000})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		redisCleanUp(client)
		os.Exit(0)
	}()

	return client
}

func redisCleanUp(conn *redis.Client) {
	log.Println("Cleaning location DB...")
	conn.FlushDb()
	conn.Close()
}
