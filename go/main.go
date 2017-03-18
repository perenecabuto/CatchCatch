package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"

	"flag"

	"github.com/garyburd/redigo/redis"
	io "github.com/googollee/go-socket.io"
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
	eventH := NewEventHandler(server, service)

	http.Handle("/socket.io/", eventH)
	http.Handle("/", http.FileServer(http.Dir("../web")))
	log.Println("Serving at localhost: 5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}

func mustRedisConnect() *redis.Pool {
	pool := redis.NewPool(func() (redis.Conn, error) {
		c, err := redis.Dial("tcp", *redisAddress)
		if err != nil {
			return nil, err
		}

		return c, err
	}, *maxConnections)

	conn, err := pool.Dial()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	if res, err := conn.Do("PING"); err != nil {
		log.Fatal(err)
	} else {
		log.Println("PING", res)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		redisCleanUp(conn)
		os.Exit(0)
	}()

	return pool
}

func redisCleanUp(conn redis.Conn) {
	log.Println("Cleaning location DB...")
	conn.Send("FLUSHDB")
	conn.Flush()
	conn.Close()
}
