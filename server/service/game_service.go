package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/perenecabuto/CatchCatch/server/service/messages"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
)

const (
	// DefaultGeoEventRange set the watcher radar radius size
	DefaultGeoEventRange = 5000

	// GameChangeTopic is the topic used for game updates on messages.Dispatcher
	GameChangeTopic = "game:update"
)

type GameService interface {
	Create(gameID, serverID string) (*game.Game, error)
	Update(g *game.Game, serverID string, evt game.Event) error
	Remove(gameID string) error
	IsGameRunning(gameID string) (bool, error)
	GameByID(gameID string) (*game.Game, *game.Event, error)

	ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, exit bool) error) error
	ObservePlayersCrossGeofences(ctx context.Context, callback func(string, model.Player) error) error
	ObserveGamesEvents(ctx context.Context, callback func(game.Game, game.Event) error) error
}

type Tile38GameService struct {
	repo     repository.Repository
	stream   repository.EventStream
	messages messages.Dispatcher
}

func NewGameService(r repository.Repository, s repository.EventStream, m messages.Dispatcher) GameService {
	return &Tile38GameService{r, s, m}
}

func (gs *Tile38GameService) Create(gameID string, serverID string) (*game.Game, error) {
	f, err := gs.repo.FeatureByID("geofences", gameID)
	if err != nil {
		return nil, err
	}
	_, err = gs.repo.SetFeature("game", gameID, f.Coordinates)
	if err != nil {
		return nil, err
	}

	game, evt := game.NewGame(gameID)
	gameEvt := &GameEvent{Game: *game, Event: evt, LastUpdate: time.Now(), ServerID: serverID}
	serialized, err := json.Marshal(gameEvt)
	if err != nil {
		return nil, err
	}
	err = gs.repo.SetFeatureExtraData("game", gameID, string(serialized))
	if err != nil {
		return nil, err
	}
	err = gs.messages.Publish(GameChangeTopic, serialized)
	if err != nil {
		return nil, err
	}
	return game, nil
}

// TODO: remove this and only check if game exists
func (gs *Tile38GameService) IsGameRunning(gameID string) (bool, error) {
	gameEvt, err := gs.findGameEvent(gameID)
	if err != nil {
		return false, err
	}
	lastUpdate := gameEvt.LastUpdate
	expiration := lastUpdate.Add(20 * time.Second)
	return time.Now().Before(expiration), nil
}

func (gs *Tile38GameService) Update(g *game.Game, serverID string, evt game.Event) error {
	gameEvt := &GameEvent{Game: *g, Event: evt, LastUpdate: time.Now(), ServerID: serverID}
	serialized, err := json.Marshal(gameEvt)
	if err != nil {
		return err
	}
	err = gs.repo.SetFeatureExtraData("game", g.ID, string(serialized))
	if err != nil {
		return err
	}
	return gs.messages.Publish(GameChangeTopic, serialized)
}

func (gs *Tile38GameService) findGameEvent(gameID string) (*GameEvent, error) {
	data, err := gs.repo.FeatureExtraData("game", gameID)
	if err != nil {
		return nil, err
	}
	gameEvt := GameEvent{}
	err = json.Unmarshal([]byte(data), &gameEvt)
	return &gameEvt, err
}

var GameEventNotFound = repository.ErrFeatureNotFound

func (gs *Tile38GameService) GameByID(gameID string) (*game.Game, *game.Event, error) {
	gameEvt, err := gs.findGameEvent(gameID)
	if err != nil {
		return nil, nil, err
	}

	started := gameEvt.Game.Started()
	players := gameEvt.Game.Players()
	targetID := gameEvt.Game.TargetID()
	evt := gameEvt.Event
	return game.NewGameWithParams(gameID, started, players, targetID), &evt, nil
}

func (gs *Tile38GameService) Remove(gameID string) error {
	if err := gs.repo.DelFeatureExtraData("game", gameID); err != nil {
		return err
	}
	return gs.repo.RemoveFeature("game", gameID)
}

func (gs *Tile38GameService) ObservePlayersCrossGeofences(ctx context.Context, callback func(string, model.Player) error) error {
	return gs.stream.StreamNearByEvents(ctx, "player", "geofences", "*", 0, func(d *repository.Detection) error {
		gameID := d.NearByFeatID
		if gameID == "" {
			return nil
		}
		p := model.Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		return callback(gameID, p)
	})
}

func (gs *Tile38GameService) ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, exit bool) error) error {
	return gs.stream.StreamIntersects(ctx, "player", "game", gameID, func(d *repository.Detection) error {
		p := model.Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		return callback(p, d.Intersects == repository.Exit)
	})
}

func (gs *Tile38GameService) ObserveGamesEvents(ctx context.Context, callback func(game.Game, game.Event) error) error {
	return gs.messages.Subscribe(GameChangeTopic, func(data []byte) error {
		gameEvt := GameEvent{}
		err := json.Unmarshal(data, &gameEvt)
		if err != nil {
			return err
		}
		return callback(gameEvt.Game, gameEvt.Event)
	})
}

type GameEvent struct {
	Game       game.Game
	Event      game.Event
	LastUpdate time.Time
	ServerID   string
}
