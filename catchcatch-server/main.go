package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"time"

	zconf "github.com/grandcat/zeroconf"
	redis "gopkg.in/redis.v5"
)

var (
	tile38Addr     = flag.String("tile38-addr", "localhost:9851", "redis address")
	maxConnections = flag.Int("tile38-connections", 100, "tile38 address")
	port           = flag.Int("port", 5000, "server port")
	webDir         = flag.String("web-dir", "../web", "web files dir")
	zconfEnabled   = flag.Bool("zconf", false, "start zeroconf server")
	debugMode      = flag.Bool("debug", false, "debug")
	wsdriver       = flag.String("wsdriver", "xnet", "options: xnet, gobwas")

	influxdbAddr = flag.String("influxdb-addr", "http://localhost:8086", "influxdb address")
	influxdbDB   = flag.String("influxdb-db", "catchcatch", "influxdb database name")
	influxdbUser = flag.String("influxdb-user", "", "influxdb user")
	influxdbPass = flag.String("influxdb-pass", "", "influxdb password")
)

func main() {
	flag.Parse()
	if *zconfEnabled {
		zcServer, _ := zconf.Register("CatchCatch", "_catchcatch._tcp", "", *port, nil, nil)
		defer zcServer.Shutdown()
	}

	metrics, err := NewMetricsCollector(*influxdbAddr, *influxdbDB, *influxdbUser, *influxdbPass)
	if err != nil {
		log.Panic(err)
	}
	if err := metrics.RunGlobalCollector(); err != nil {
		log.Panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream := NewEventStream(*tile38Addr)
	client := mustConnectTile38(*debugMode)
	repo := NewRepository(client)
	playerService := NewPlayerLocationService(repo)
	wsHandler := selectWsDriver(*wsdriver)
	server := NewWSServer(wsHandler)
	watcher := NewGameWatcher(stream, server)
	onExit(func() {
		cancel()
		client.Close()
		server.CloseAll()
	})

	go func() {
		if err := watcher.WatchGamesForever(ctx); err != nil {
			log.Panic("WatchGamesForever:error", err)
		}
	}()
	go watcher.WatchCheckpoints(ctx)
	go watcher.WatchGeofences(ctx)
	go watcher.WatchPlayers(ctx)

	eventH := NewEventHandler(server, playerService)
	server.OnConnected(eventH.onConnection)
	http.Handle("/ws", recoverWrapper(server.Listen(ctx)))
	http.Handle("/", http.FileServer(http.Dir(*webDir)))

	log.Println("Serving at localhost:", strconv.Itoa(*port), "...")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func selectWsDriver(name string) WSDriver {
	switch name {
	case "gobwas":
		return NewGobwasWSDriver()
	default:
		return NewXNetWSDriver()
	}
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

func recoverWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := withRecover(func() error {
			h.ServeHTTP(w, r)
			return nil
		})
		if err != nil {
			http.Error(w, "", http.StatusInternalServerError)
		}
	})
}

func withRecover(fn func() error) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = fmt.Errorf("%v", r)
			log.Printf("[panic withRecover] %v", err)
			debug.PrintStack()
		}
	}()
	return fn()
}

func onExit(fn func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		<-c
		fn()
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
}
