package main

import (
	"context"
	"log"
	"runtime/debug"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

// WatchCheckpoints ...
func (gw *GameWatcher) WatchCheckpoints(ctx context.Context) {
	err := gw.stream.StreamNearByEvents(ctx, "player", "checkpoint", "*", 1000, func(d *Detection) error {
		if d.Intersects == Exit {
			return nil
		}

		payload := &protobuf.Detection{
			EventName:    proto.String("checkpoint:detected"),
			Id:           &d.FeatID,
			FeatId:       &d.FeatID,
			Lon:          &d.Lon,
			Lat:          &d.Lat,
			NearByFeatId: &d.NearByFeatID,
			NearByMeters: &d.NearByMeters,
		}
		return gw.wss.Emit(d.FeatID, payload)
	})
	if err != nil {
		log.Println("Error to stream geofence:event", err)
		debug.PrintStack()
	}
}
