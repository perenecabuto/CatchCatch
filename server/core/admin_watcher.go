package core

import (
	"context"
	"log"

	"github.com/golang/protobuf/proto"

	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/protobuf"
	"github.com/perenecabuto/CatchCatch/server/service"
	"github.com/perenecabuto/CatchCatch/server/websocket"
)

// AdminWatcher is responsable to watch and notify admins about game events
type AdminWatcher struct {
	service service.GeoFeatureService
	wss     *websocket.WSServer
}

// NewAdminWatcher creates a new game watcher
func NewAdminWatcher(service service.GeoFeatureService, wss *websocket.WSServer) *AdminWatcher {
	return &AdminWatcher{service, wss}
}

// WatchPlayers observe players around players and notify it's position
func (w *AdminWatcher) WatchPlayers(ctx context.Context) error {
	return w.service.ObservePlayersAround(ctx, func(playerID string, remotePlayer model.Player, exit bool) error {
		evtName := proto.String("remote-player:updated")
		if exit {
			w.wss.Close(remotePlayer.ID)
			evtName = proto.String("remote-player:destroy")
		}
		err := w.wss.Broadcast(&protobuf.Player{EventName: evtName,
			Id: &remotePlayer.ID, Lon: &remotePlayer.Lon, Lat: &remotePlayer.Lat})
		if err != websocket.ErrWSConnectionNotFound && err != nil {
			log.Println("remote-player:updated error", err.Error())
		}
		return nil
	})
}

// WatchGeofences watch for geofences events and notify players around
func (w *AdminWatcher) WatchGeofences(ctx context.Context) error {
	return w.service.ObservePlayerNearToFeature(ctx, "geofences", func(playerID string, distTo float64, f model.Feature) error {
		err := w.wss.Emit(playerID,
			&protobuf.Feature{
				EventName: proto.String("admin:feature:added"), Id: &f.ID,
				Group: proto.String("geofences"), Coords: &f.Coordinates})
		if err != nil {
			log.Println("AdminWatcher:WatchGeofences:", err.Error())
		}
		return nil
	})
}

// WatchCheckpoints ...
func (w *AdminWatcher) WatchCheckpoints(ctx context.Context) error {
	return w.service.ObservePlayerNearToFeature(ctx, "checkpoint", func(playerID string, distTo float64, f model.Feature) error {
		payload := &protobuf.Detection{
			EventName:    proto.String("checkpoint:detected"),
			Id:           &f.ID,
			FeatId:       &f.ID,
			NearByFeatId: &playerID,
			NearByMeters: &distTo,
		}
		if err := w.wss.Emit(playerID, payload); err != nil {
			log.Println("AdminWatcher:WatchCheckpoints:", err.Error())
		}
		return nil
	})
}
