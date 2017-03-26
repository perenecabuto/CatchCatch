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
	tile38Addr     = flag.String("tile38-addr", "localhost:9851", "redis address")
	maxConnections = flag.Int("tile38-connections", 100, "tile38 address")
	port           = flag.Int("port", 8888, "server port")
)

func main() {
	flag.Parse()

	client := mustConnectTile38()
	service := &PlayerLocationService{client}
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			log.Println("TILE38 DEBUG:", cmd.String())
			return oldProcess(cmd)
		}
	})

	go func() {
		err := service.StreamGeofenceEvents(*tile38Addr, func(msg string) {
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

func mustConnectTile38() *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: *tile38Addr, PoolSize: 1000, DialTimeout: 1 * time.Second})

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		tile38Cleanup(client)
		os.Exit(0)
	}()

	return client
}

func tile38Cleanup(conn *redis.Client) {
	log.Println("Cleaning location DB...")
	conn.FlushDb()
	conn.Close()
}
