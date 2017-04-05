package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
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
)

func main() {
	flag.Parse()

	if *zconfEnabled {
		zcServer, _ := zconf.Register("CatchCatch", "_catchcatch._tcp", "", *port, nil, nil)
		defer zcServer.Shutdown()
	}

	client := mustConnectTile38()
	service := &PlayerLocationService{client}
	client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			log.Println("TILE38 DEBUG:", cmd.String())
			return oldProcess(cmd)
		}
	})

	server, err := io.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	server.On("error", func(so io.Socket, err error) {
		log.Println("error:", err)
	})

	sessions := NewSessionManager()
	stream := NewEventStream(*tile38Addr)

	go handleGames(stream, sessions, service)
	go handleCheckointsDetection(stream, sessions, server)

	eventH := NewEventHandler(server, service, sessions)

	http.Handle("/ws/", eventH)
	http.Handle("/", http.FileServer(http.Dir(*webDir)))
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

func handleCheckointsDetection(stream EventStream, sessions *SessionManager, server *io.Server) {
	err := stream.StreamNearByEvents("player", "checkpoint", 1000, func(d *Detection) {
		payload, _ := json.Marshal(d)
		if err := sessions.Emit(d.FeatID, "checkpoint:detected", string(payload)); err != nil {
			log.Println("Error to notify player", d.FeatID, err)
		}
		server.BroadcastTo("main", "admin:feature:checkpoint", d)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
	}
}
