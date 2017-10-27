package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/model"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

var addr = flag.String("addr", "localhost:5000", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/ws"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})
	player := model.Player{}

	go func() {
		defer c.Close()
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			msg := &protobuf.Simple{}
			if err := proto.Unmarshal(message, msg); err != nil {
				log.Println("readMessage(unmarshall): ", err.Error(), message)
				continue
			}

			switch *msg.EventName {
			case "player:registered":
				p := &protobuf.Player{}
				if err := proto.Unmarshal(message, p); err != nil {
					log.Println("error parsing player: ", err.Error(), p)
					continue
				}

				log.Println("player: ", p)
				player.ID, player.Lat, player.Lon = *p.Id, *p.Lat, *p.Lon
				player.Lat, player.Lon = -30.03495, -51.21866
			}
			log.Printf("recv: %s", msg)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		player.Lat += 0.00001
		player.Lon += 0.00001
		select {
		case <-ticker.C:
			evt := "player:update"
			msg := &protobuf.Player{EventName: &evt, Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat}
			payload, _ := proto.Marshal(msg)
			err := c.WriteMessage(websocket.BinaryMessage, payload)
			if err != nil {
				log.Println("write:", err)
				return
			}
		case <-interrupt:
			log.Println("closing")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			c.Close()
			return
		}
	}
}
