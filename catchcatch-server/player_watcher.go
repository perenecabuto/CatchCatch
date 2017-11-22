package main

import (
	"context"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

// WatchPlayers observe players around players and notify it's position
func (gw *GameWatcher) WatchPlayers(ctx context.Context) error {
	return gw.stream.StreamNearByEvents(ctx, "player", "player", "*", 5000, func(d *Detection) error {
		playerID, remotePlayerID := d.NearByFeatID, d.FeatID
		switch d.Intersects {
		case Inside:
			err := gw.wss.Emit(playerID, &protobuf.Player{EventName: proto.String("remote-player:updated"),
				Id: &remotePlayerID, Lon: &d.Lon, Lat: &d.Lat})
			if err != ErrWSConnectionNotFound && err != nil {
				log.Println("remote-player:updated error", err.Error())
			}
		case Exit:
			gw.wss.Close(remotePlayerID)
			err := gw.wss.Broadcast(&protobuf.Player{EventName: proto.String("remote-player:destroy"),
				Id: &remotePlayerID, Lon: &d.Lon, Lat: &d.Lat})
			if err != nil {
				log.Println("remote-player:updated error", err.Error())
			}
		}
		return nil
	})
}
