package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"

	redis "github.com/go-redis/redis"
	zconf "github.com/grandcat/zeroconf"
	nats "github.com/nats-io/go-nats"
	uuid "github.com/satori/go.uuid"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/execfunc"
	"github.com/perenecabuto/CatchCatch/server/metrics"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
	"github.com/perenecabuto/CatchCatch/server/websocket"
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
	stream := repository.NewEventStream(*tile38Addr)
	client := mustConnectTile38(*debugMode)
	repo := repository.NewRepository(client)
	playerService := service.NewPlayerLocationService(repo, stream)

	natsConn := mustConnectNats(nats.DefaultURL)
	dispatcher := messages.NewNatsDispatcher(natsConn)
	gameService := service.NewGameService(repo, stream, dispatcher)
	featService := service.NewGeoFeatureService(repo)
	wsHandler := selectWsDriver(*wsdriver)
	server := websocket.NewWSServer(wsHandler)
	aWatcher := core.NewAdminWatcher(playerService, server)
	gWatcher := core.NewGameWatcher(*serverID, gameService, server)
	worker := core.NewGameWorker(*serverID, gameService)
	execfunc.OnExit(func() {
		cancel()
		client.Close()
		server.CloseAll()
	})

	startInBG(ctx,
		worker.WatchGames,
		gWatcher.WatchGameEvents,
		aWatcher.WatchCheckpoints,
		aWatcher.WatchGeofences,
		aWatcher.WatchPlayers,
	)

	eventH := core.NewEventHandler(server, playerService, featService)
	server.SetEventHandler(eventH)
	http.Handle("/ws", execfunc.RecoverWrapper(server.Listen(ctx)))
	http.Handle("/", http.FileServer(http.Dir(*webDir)))

	log.Println("Serving at localhost:", strconv.Itoa(*port), "...")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func selectWsDriver(name string) websocket.WSDriver {
	switch name {
	case "gobwas":
		return websocket.NewGobwasWSDriver()
	default:
		return websocket.NewXNetWSDriver()
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

func mustConnectNats(url string) *nats.Conn {
	conn, err := nats.Connect(url)
	if err != nil {
		log.Panic("Nat connection:", err)
	}
	return conn
}

func startInBG(ctx context.Context, funcs ...func(context.Context) error) {
	for _, fn := range funcs {
		go func(ctx context.Context, fn func(context.Context) error) {
			err := fn(ctx)
			if err != nil {
				fnname := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
				log.Fatalf("Background Task<%s> error: %s", fnname, err)
			}
		}(ctx, fn)
	}
}
