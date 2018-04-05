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

// FeatureEventNearToAdmin notifies the AdminHandler of feature events
type FeatureEventNearToAdmin interface {
	OnFeatureEventNearToAdmin(
		context.Context,
		func(adminID string, feat model.Feature, action string) error,
	) error
}

// AdminHandler handle websocket events
type AdminHandler struct {
	players service.PlayerLocationService
	watcher FeatureEventNearToAdmin

	wss *websocket.WSServer
}

// NewAdminHandler AdminHandler builder
func NewAdminHandler(p service.PlayerLocationService, w FeatureEventNearToAdmin) *AdminHandler {
	handler := &AdminHandler{players: p, watcher: w}
	return handler
}

// OnStart observe feature events for admins
func (h *AdminHandler) OnStart(ctx context.Context, wss *websocket.WSServer) error {
	h.wss = wss

	return h.watcher.OnFeatureEventNearToAdmin(ctx,
		func(adminID string, feat model.Feature, action string) error {
			err := wss.Emit(adminID, &protobuf.Feature{
				EventName: proto.String("admin:feature:" + action),
				Group:     &feat.Group, Id: &feat.ID, Coords: &feat.Coordinates})
			if err != websocket.ErrWSConnectionNotFound {
				// TODO: add servers broadcast ping
				// if no server responds with user found
				// remote this admin
				return nil
			}
			if err != nil {
				log.Println("[AdminHandler] WatchFeatureEvents error:", err)
			}
			return nil
		})
}

// OnConnection handles game and admin connection events
func (h *AdminHandler) OnConnection(ctx context.Context, c *websocket.WSConnectionHandler) error {
	log.Println("[AdminHandler] [admin] connected", c.ID)

	lat, lon := 0.0, 0.0
	h.players.SetAdmin(c.ID, lat, lon)

	c.OnDisconnected(func() {
		h.players.RemoveAdmin(c.ID)
		log.Println("[AdminHandler] [admin] disconnected", c.ID)
	})

	c.On("admin:position:update", h.onUpdatePosition(c))
	c.On("admin:players:disconnect", h.onDisconnectPlayer(c))
	c.On("admin:players:request", h.onRequestPlayers(c))
	c.On("admin:feature:request", h.onRequestFeatures(c))
	c.On("admin:feature:add", h.onAddFeature())
	c.On("admin:clear", h.onClear())

	return nil
}

func (h *AdminHandler) onUpdatePosition(so *websocket.WSConnectionHandler) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Player{}
		proto.Unmarshal(buf, msg)
		log.Println("admin position", msg)
		h.players.SetAdmin(so.ID, msg.GetLat(), msg.GetLon())
	}
}

func (h *AdminHandler) onRequestPlayers(so *websocket.WSConnectionHandler) func([]byte) {
	return func([]byte) {
		players, err := h.players.All()
		if err != nil {
			log.Println("[AdminHandler] player:request-remotes event error: " + err.Error())
		}
		event := "remote-player:new"
		for _, p := range players {
			if p == nil {
				continue
			}
			err := so.Emit(&protobuf.Player{EventName: &event, Id: &p.ID, Lon: &p.Lon, Lat: &p.Lat})
			if err != nil {
				log.Println("[AdminHandler] player:request-remotes event error: " + err.Error())
			}
		}
	}
}

func (h *AdminHandler) onDisconnectPlayer(c *websocket.WSConnectionHandler) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Simple{}
		proto.Unmarshal(buf, msg)
		playerID := msg.GetId()
		log.Println("[AdminHandler] admin:players:disconnect", playerID)
		err := h.players.Remove(playerID)
		if err == service.ErrFeatureNotFound {
			log.Println("[AdminHandler] admin:players:disconnect: player not found - ", playerID)
		}
		h.wss.Broadcast(&protobuf.Simple{EventName: proto.String("admin:players:disconnected"), Id: &playerID})
	}
}

func (h *AdminHandler) onClear() func([]byte) {
	return func([]byte) {
		// TODO: send this message by broaker
		// h.games.Clear()
		h.players.Clear()
		h.wss.CloseAll()
	}
}

func (h *AdminHandler) onAddFeature() func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		var err error
		switch msg.GetGroup() {
		case "geofences":
			err = h.players.SetGeofence(msg.GetId(), msg.GetCoords())
		case "checkpoint":
			err = h.players.SetCheckpoint(msg.GetId(), msg.GetCoords())
		}
		if err != nil {
			log.Println("[AdminHandler] Error to add feature:", err)
		}
	}
}

func (h *AdminHandler) onRequestFeatures(c *websocket.WSConnectionHandler) func([]byte) {
	return func(buf []byte) {
		msg := &protobuf.Feature{}
		proto.Unmarshal(buf, msg)

		features, err := h.players.Features()
		if err != nil {
			log.Println("[AdminHandler] Error on sendFeatures:", err)
		}
		event := "admin:feature:added"
		for _, f := range features {
			c.Emit(&protobuf.Feature{EventName: &event, Id: &f.ID, Group: &f.Group, Coords: &f.Coordinates})
		}
	}
}
