package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"strconv"
	"time"

	zconf "github.com/grandcat/zeroconf"
	uuid "github.com/satori/go.uuid"
	redis "gopkg.in/redis.v5"

	"github.com/perenecabuto/CatchCatch/catchcatch-server/execfunc"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/metrics"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/service"
)

var (
	serverID       = flag.String("id", uuid.NewV4().String(), "server id")
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

	metrics, err := metrics.NewMetricsCollector(*influxdbAddr, *influxdbDB, *influxdbUser, *influxdbPass)
	if err != nil {
		log.Panic(err)
	}
	if err := metrics.RunGlobalCollector(); err != nil {
		log.Panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	stream := service.NewEventStream(*tile38Addr)
	client := mustConnectTile38(*debugMode)
	repo := service.NewRepository(client)
	playerService := service.NewPlayerLocationService(repo)
	gameService := service.NewGameService(repo, stream)
	featService := service.NewGeoFeatureService(repo, stream)
	wsHandler := selectWsDriver(*wsdriver)
	server := NewWSServer(wsHandler)
	aWatcher := NewAdminWatcher(featService, server)
	gWatcher := NewGameWatcher(*serverID, gameService, server)
	worker := NewGameWorker(*serverID, gameService)
	execfunc.OnExit(func() {
		cancel()
		client.Close()
		server.CloseAll()
	})

	go worker.WatchGames(ctx)
	go gWatcher.WatchGameEventsForever(ctx)
	go aWatcher.WatchCheckpoints(ctx)
	go aWatcher.WatchGeofences(ctx)
	go aWatcher.WatchPlayers(ctx)

	eventH := NewEventHandler(server, playerService, featService)
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
