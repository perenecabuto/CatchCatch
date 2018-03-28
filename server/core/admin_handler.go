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

//TODO: set game status on db

// AdminHandler handle websocket events
type AdminHandler struct {
	server  *websocket.WSServer
	players service.PlayerLocationService
	geo     service.GeoFeatureService
}

// NewAdminHandler AdminHandler builder
func NewAdminHandler(server *websocket.WSServer, players service.PlayerLocationService, geo service.GeoFeatureService) *AdminHandler {
	handler := &AdminHandler{server, players, geo}
	return handler
}

// Event handlers

// OnConnection handles game and admin connection events
func (h *AdminHandler) OnConnection(c *websocket.WSConnListener) {
	log.Println("new admin connected", c.ID)
	c.On("admin:disconnect", h.onDisconnectByID(c))
	c.On("admin:feature:add", h.onAddFeature())
	c.On("admin:feature:request-remotes", h.onPlayerRequestRemotes(c))
	c.On("admin:feature:request-list", h.onRequestFeatures(c))
	c.On("admin:clear", h.onClear())
}

func (h *AdminHandler) onPlayerRequestRemotes(so *websocket.WSConnListener) func([]byte) {
	return func([]byte) {
		players, err := h.players.All()
		if err != nil {
			log.Println("player:request-remotes event error: " + err.Error())
		}
		event := "remote-player:new"
		for _, p := range players {
			if p == nil {
				continue
			}
			err := so.Emit(&protobuf.Player{EventName: &event, Id: &p.ID, Lon: &p.Lon, Lat: &p.Lat})
			if err != nil {
				log.Println("player:request-remotes event error: " + err.Error())
			}
		}
	}
}

func (h *AdminHandler) onDisconnectByID(c *websocket.WSConnListener) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Simple{}
		proto.Unmarshal(buf, msg)
		log.Println("admin:disconnect", msg.GetId())
		player := &model.Player{ID: msg.GetId()}
		err := h.players.Remove(player)
		if err == service.ErrFeatureNotFound {
			// Notify remote-player removal to ghost players on admin
			log.Println("admin:disconnect:force", msg.GetId())
			c.Emit(&protobuf.Player{EventName: proto.String("remote-player:destroy"),
				Id: &player.ID, Lon: &player.Lon, Lat: &player.Lat})
		}
		h.server.Remove(msg.GetId())
	}
}

func (h *AdminHandler) onClear() func([]byte) {
	return func([]byte) {
		// TODO: send this message by broaker
		// h.games.Clear()
		h.geo.Clear()
		h.server.CloseAll()
	}
}

// Map events

func (h *AdminHandler) onAddFeature() func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		err := h.geo.SetFeature(msg.GetGroup(), msg.GetId(), msg.GetCoords())
		if err != nil {
			log.Println("Error to create feature:", err)
			return
		}
	}
}

func (h *AdminHandler) onRequestFeatures(c *websocket.WSConnListener) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		features, err := h.geo.FeaturesByGroup(msg.GetGroup())
		if err != nil {
			log.Println("Error on sendFeatures:", err)
		}
		event := "admin:feature:added"
		for _, f := range features {
			c.Emit(&protobuf.Feature{EventName: &event, Id: &f.ID, Group: &f.Group, Coords: &f.Coordinates})
		}
	}
}

// WatchPlayers observe players around players and notify it's position
func (h *AdminHandler) WatchPlayers(ctx context.Context) error {
	return h.players.ObservePlayersAround(ctx, func(playerID string, remotePlayer model.Player, exit bool) error {
		evtName := proto.String("remote-player:updated")
		if exit {
			h.server.Close(remotePlayer.ID)
			evtName = proto.String("remote-player:destroy")
		}
		err := h.server.Broadcast(&protobuf.Player{EventName: evtName,
			Id: &remotePlayer.ID, Lon: &remotePlayer.Lon, Lat: &remotePlayer.Lat})
		if err != websocket.ErrWSConnectionNotFound && err != nil {
			log.Println("remote-player:updated error", err.Error())
		}
		return nil
	})
}

// WatchGeofences watch for geofences events and notify players around
func (h *AdminHandler) WatchGeofences(ctx context.Context) error {
	return h.players.ObservePlayerNearToFeature(ctx, "geofences", func(playerID string, distTo float64, f model.Feature) error {
		err := h.server.Emit(playerID,
			&protobuf.Feature{
				EventName: proto.String("admin:feature:added"), Id: &f.ID,
				Group: proto.String("geofences"), Coords: &f.Coordinates})
		if err != nil {
			log.Println("AdminHandler:WatchGeofences:", err.Error())
		}
		return nil
	})
}
