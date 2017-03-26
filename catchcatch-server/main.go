package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"time"

	io "github.com/googollee/go-socket.io"
	redis "gopkg.in/redis.v5"
)

var (
	redisAddress   = flag.String("redis-addr", "localhost:9851", "redis address")
	maxConnections = flag.Int("redis-connections", 100, "redis address")
	port           = flag.Int("port", 8888, "server port")
)

func main() {
	flag.Parse()

	client := mustRedisConnect()
	service := &PlayerLocationService{client}
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			log.Println("REDIS DEBUG:", cmd.String())
			return oldProcess(cmd)
		}
	})

	go func() {
		err := service.StreamGeofenceEvents(addr, func(msg string) {
			log.Println("geofence:event", msg)
		})
		if err != nil {
			log.Println("Error to stream geofence:event", err)
		}
	}()

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
	log.Println("Serving at localhost:", strconv.Itoa(*port), "...")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func mustRedisConnect() *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: *redisAddress, PoolSize: 1000, DialTimeout: 1 * time.Second})

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
