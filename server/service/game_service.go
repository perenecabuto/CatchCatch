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

//TODO: set game status on db
//TODO: move messages to worker

type GameService interface {
	Create(gameID, coordinates string) (*game.Game, error)
	Update(g *game.Game, evt game.Event) error
	Remove(gameID string) error
	GameByID(gameID string) (*game.Game, *game.Event, error)
	GamesAround(p model.Player) ([]GameWithCoords, error)

	ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, exit bool) error) error
	ObserveGamesEvents(ctx context.Context, callback func(*game.Game, game.Event) error) error
}

type GameWithCoords struct {
	*game.Game
	Coords string
}

type Tile38GameService struct {
	repo     repository.Repository
	stream   repository.EventStream
	messages messages.Dispatcher
}

func NewGameService(r repository.Repository, s repository.EventStream, m messages.Dispatcher) GameService {
	return &Tile38GameService{r, s, m}
}

func (gs *Tile38GameService) Create(gameID, coordinates string) (*game.Game, error) {
	_, err := gs.repo.SetFeature("game", gameID, coordinates)
	if err != nil {
		return nil, err
	}

	game, evt := game.NewGame(gameID)
	gameEvt := &GameEvent{Game: game, Event: evt, LastUpdate: time.Now()}
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

func (gs *Tile38GameService) Update(g *game.Game, evt game.Event) error {
	gameEvt := &GameEvent{Game: g, Event: evt, LastUpdate: time.Now()}
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

// GamesAround returns a list of games with its geo coordinates
func (gs *Tile38GameService) GamesAround(p model.Player) ([]GameWithCoords, error) {
	feats, err := gs.repo.FeaturesAround("game", p.Point())
	if err != nil {
		return nil, err
	}

	games := make([]GameWithCoords, len(feats))
	for i, f := range feats {
		games[i] = GameWithCoords{
			Game:   &game.Game{ID: f.ID},
			Coords: f.Coordinates,
		}
	}

	return games, nil
}

func (gs *Tile38GameService) ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, exit bool) error) error {
	return gs.stream.StreamIntersects(ctx, "player", "game", gameID, func(d *repository.Detection) error {
		p := model.Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		return callback(p, d.Intersects == repository.Exit)
	})
}

// TODO: tirar isso daqui, por no worker ou em algum comunicador
func (gs *Tile38GameService) ObserveGamesEvents(ctx context.Context, callback func(*game.Game, game.Event) error) error {
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
	Game       *game.Game
	Event      game.Event
	LastUpdate time.Time
}
