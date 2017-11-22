package main

import (
	"context"
	"log"

	"github.com/golang/protobuf/proto"
	"github.com/perenecabuto/CatchCatch/catchcatch-server/protobuf"
)

func (gw *GameWatcher) WatchGeofences(ctx context.Context) error {
	// TODO: only notify admins about new geofences
	return gw.stream.StreamNearByEvents(ctx, "geofences", "player", "*", 5000, func(d *Detection) error {
		switch d.Intersects {
		case Inside:
			coords := `{"type":"Polygon","coordinates":` + d.Coordinates + "}"
			err := gw.wss.Emit(d.NearByFeatID, &protobuf.Feature{EventName: proto.String("admin:feature:added"), Id: &d.FeatID,
				Group: proto.String("geofences"), Coords: &coords})
			if err != nil {
				log.Println("admin:feature:added error", err.Error())
			}
		}
		return nil
	})
}
