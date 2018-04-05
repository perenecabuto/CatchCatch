package main

import (
	"context"
	"flag"
	"log"
	"net/http"
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
	"github.com/perenecabuto/CatchCatch/server/worker"
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

	workerRedisAddr = flag.String("workers-redis-addr", "localhost:6379", "distributed workers' redis address")

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
	tile38Cli := mustConnectRedis(*tile38Addr, *debugMode)
	repo := repository.NewRepository(tile38Cli)
	playerService := service.NewPlayerLocationService(repo, stream)

	natsConn := mustConnectNats(nats.DefaultURL)
	dispatcher := messages.NewNatsDispatcher(natsConn)
	gameService := service.NewGameService(repo, stream, dispatcher)
	wsdriver := selectWsDriver(*wsdriver)

	workersCli := mustConnectRedis(*workerRedisAddr, *debugMode)
	workersCli.FlushAll()
	workers := worker.NewGoredisWorkerManager(workersCli)

	gameWorker := core.NewGameWorker(gameService)
	geofenceEventsWorker := core.NewGeofenceEventsWorker(playerService, workers)
	checkpointWatcher := core.NewCheckpointWatcher(dispatcher, playerService)
	featuresWatcher := core.NewFeaturesEventsWatcher(dispatcher, playerService)

	workers.Add(gameWorker)
	workers.Add(geofenceEventsWorker)
	workers.Add(checkpointWatcher)
	workers.Add(featuresWatcher)

	workers.Start(ctx)
	// TODO: verify server id on these workers
	workers.RunUnique(geofenceEventsWorker, worker.TaskParams{"serverID": serverID})
	workers.RunUnique(checkpointWatcher, worker.TaskParams{"serverID": serverID})
	workers.RunUnique(featuresWatcher, worker.TaskParams{"serverID": serverID})

	playerH := core.NewPlayerHandler(playerService, gameWorker)
	playersConnections := websocket.NewWSServer(wsdriver, playerH)
	adminH := core.NewAdminHandler(playerService, featuresWatcher)
	adminConnections := websocket.NewWSServer(wsdriver, adminH)

	http.Handle("/admin", execfunc.RecoverWrapper(adminConnections.Listen(ctx)))
	http.Handle("/player", execfunc.RecoverWrapper(playersConnections.Listen(ctx)))
	http.Handle("/", http.FileServer(http.Dir(*webDir)))

	execfunc.OnExit(func() {
		cancel()
		workers.Stop()
		tile38Cli.Close()
		workersCli.Close()
		adminConnections.CloseAll()
		playersConnections.CloseAll()
	})

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

func mustConnectRedis(addr string, debug bool) *redis.Client {
	client := redis.NewClient(&redis.Options{Addr: addr, PoolSize: 1000, DialTimeout: 1 * time.Second})
	if debug {
		client.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
			return func(cmd redis.Cmder) error {
				log.Printf("[REDIS(%s)]: %s", addr, cmd.String())
				return oldProcess(cmd)
			}
		})
	}
	return client
}

func mustConnectNats(url string) *nats.Conn {
	conn, err := nats.Connect(url)
	if err != nil {
		log.Panic("Nat connection:", err)
	}
	return conn
}
