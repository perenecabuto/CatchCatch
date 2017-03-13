package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/garyburd/redigo/redis"
	socketio "github.com/googollee/go-socket.io"
)

func main() {
	conn := mustRedisConnect()
	service := &PlayerLocationService{conn}
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	server.On("connection", func(so socketio.Socket) {
		channel := "main"
		player := &Player{so.Id(), 0, 0}

		so.Join(channel)
		service.Register(player)
		so.Emit("player:registred", player)
		so.BroadcastTo(channel, "player:new", player)

		if players, err := service.All(); err == nil {
			log.Println("send players to", player)
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
			so.Emit("player:updated", player)
			so.BroadcastTo(channel, "player:updated", player)
			service.Update(player)
		})

		so.On("disconnection", func() {
			so.Leave(channel)
			so.BroadcastTo(channel, "player:destroy", player)
			service.Remove(player)
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

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		redisCleanUp(conn)
		os.Exit(0)
	}()

	return conn
}

func redisCleanUp(conn redis.Conn) {
	log.Println("Cleaning location DB...")
	conn.Send("FLUSHDB")
	conn.Flush()
	conn.Close()
}
