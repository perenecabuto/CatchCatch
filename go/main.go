package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/garyburd/redigo/redis"
	socketio "github.com/googollee/go-socket.io"
)

var (
	conn redis.Conn
)

func main() {
	conn = mustRedisConnect()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		redisCleanUp()
		os.Exit(0)
	}()

	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.On("connection", func(so socketio.Socket) {
		channel := "main"
		player := &Player{so.Id(), 0, 0}

		registerPlayer(player)
		so.Join(channel)
		log.Println("connected player", player.ID, "to channel", channel)

		so.BroadcastTo(channel, "player:new", player)
		if players, err := allPlayers(); err == nil {
			log.Println("send players to", players)
			so.Emit("player:list", players)
		} else {
			log.Println("--> error to get players", err)
		}

		so.On("player:update", func(msg string) {
			log.Println("player:update", msg)

			if err := json.Unmarshal([]byte(msg), player); err != nil {
				log.Println("player:update event error", err.Error())
				return
			}
			so.BroadcastTo(channel, "player:updated", player)
			updatePlayerPosition(player)
		})

		so.On("disconnection", func() {
			so.Leave(channel)
			so.BroadcastTo(channel, "player:destroy", player)
			removePlayer(player)
			log.Println("diconnected", player)
		})
	})

	http.Handle("/socket.io/", server)
	http.Handle("/", http.FileServer(http.Dir("../web")))
	log.Println("Serving at localhost: 5000...")
	log.Fatal(http.ListenAndServe(":5000", nil))
}

func mustRedisConnect() redis.Conn {
	conn, err := redis.Dial("tcp", "localhost:9851")
	if err != nil {
		log.Fatal(err)
	}
	if res, err := conn.Do("PING"); err != nil {
		log.Fatal(err)
	} else {
		log.Println("PING", res)
	}
	return conn
}

func redisCleanUp() {
	log.Println("Cleaning location DB...")
	conn.Send("FLUSHDB")
	conn.Flush()
	conn.Close()
}

// Player payload
type Player struct {
	ID string  `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
}

func (p *Player) String() string {
	return fmt.Sprintln("id:", p.ID, "x:", p.X, "y:", p.Y)
}

// PlayerList payload for list of players
type PlayerList struct {
	Players []*Player `json:"players"`
}

func registerPlayer(p *Player) error {
	return conn.Send("SET", "player", p.ID, "POINT", p.X, p.Y)
}

func updatePlayerPosition(p *Player) error {
	return conn.Send("SET", "player", p.ID, "POINT", p.X, p.Y)
}

func removePlayer(p *Player) error {
	return conn.Send("DEL", "player", p.ID)
}

func allPlayers() (*PlayerList, error) {
	result, err := conn.Do("SCAN", "player")
	if err != nil {
		return nil, err
	}

	var payload []interface{}
	redis.Scan(result.([]interface{}), nil, &payload)

	list := make([]*Player, len(payload))
	for i, d := range payload {
		var id string
		var data []byte
		redis.Scan(d.([]interface{}), &id, &data)
		geo := &struct {
			Coords [2]float32 `json:"coordinates"`
		}{}
		json.Unmarshal(data, geo)
		list[i] = &Player{ID: id, X: geo.Coords[0], Y: geo.Coords[1]}
	}
	return &PlayerList{list}, nil
}

func getPlayerPosition(name string) (*Player, error) {
	result, err := conn.Do("GET", "player", name)
	if err != nil {
		return nil, err
	}

	log.Println("->", string(result.([]uint8)))
	return &Player{}, nil
}
