package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	io "github.com/googollee/go-socket.io"
	zconf "github.com/grandcat/zeroconf"
	redis "gopkg.in/redis.v5"
)

var (
	tile38Addr     = flag.String("tile38-addr", "localhost:9851", "redis address")
	maxConnections = flag.Int("tile38-connections", 100, "tile38 address")
	port           = flag.Int("port", 5000, "server port")
	webDir         = flag.String("web-dir", "../web", "web files dir")
	zconfEnabled   = flag.Bool("zconf", false, "start zeroconf server")
	debug          = flag.Bool("debug", false, "debug")
)

func main() {
	flag.Parse()
	if *zconfEnabled {
		zcServer, _ := zconf.Register("CatchCatch", "_catchcatch._tcp", "", *port, nil, nil)
		defer zcServer.Shutdown()
	}

	sessions := NewSessionManager()
	stream := NewEventStream(*tile38Addr)
	client := mustConnectTile38(*debug)
	defer client.Close()

	service := &PlayerLocationService{client}
	server, err := io.NewServer(nil)
	if err != nil {
		log.Fatal("Could not start WS server", err)
	}
	server.On("error", func(so io.Socket, err error) {
		log.Println("WS error:", err)
	})

	go handleGames(stream, sessions)
	go handleCheckointsDetection(stream, sessions, server)

	eventH := NewEventHandler(server, service, sessions)
	http.Handle("/ws/", eventH)
	http.Handle("/", http.FileServer(http.Dir(*webDir)))

	log.Println("Serving at localhost:", strconv.Itoa(*port), "...")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func mustConnectTile38(debug bool) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: *tile38Addr, PoolSize: 1000, DialTimeout: 1 * time.Second})
	if debug {
		client.WrapProcess(tile38DebugWrapper)
	}

	return client
}

func tile38DebugWrapper(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
	return func(cmd redis.Cmder) error {
		log.Println("TILE38 DEBUG:", cmd.String())
		return oldProcess(cmd)
	}
}
