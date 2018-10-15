package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
)

const (
	// DefaultGeoEventRange set the watcher radar radius size
	DefaultGeoEventRange = 5000
)

type GamePlayerMove int

const (
	GamePlayerMoveInside GamePlayerMove = iota
	GamePlayerMoveOutside
)

type GameService interface {
	Create(gameID, coordinates string) (*GameWithCoords, error)
	Update(g *GameWithCoords) error
	Remove(gameID string) error
	GameByID(gameID string) (*GameWithCoords, error)
	GamesAround(p model.Player) ([]GameWithCoords, error)

	ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, exit GamePlayerMove) error) error
}

type GameWithCoords struct {
	*game.Game
	Coords string
}

type Tile38GameService struct {
	repo   repository.Repository
	stream repository.EventStream
}

func NewGameService(r repository.Repository, s repository.EventStream) GameService {
	return &Tile38GameService{r, s}
}

func (gs *Tile38GameService) Create(gameID, coordinates string) (*GameWithCoords, error) {
	_, err := gs.repo.SetFeature("game", gameID, coordinates)
	if err != nil {
		return nil, err
	}
	g := &GameWithCoords{Game: game.NewGame(gameID), Coords: coordinates}
	serialized, err := json.Marshal(g.Game)
	if err != nil {
		return nil, err
	}
	err = gs.repo.SetFeatureExtraData("game", gameID, string(serialized))
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (gs *Tile38GameService) Update(g *GameWithCoords) error {
	serialized, err := json.Marshal(g.Game)
	if err != nil {
		return err
	}
	err = gs.repo.SetFeatureExtraData("game", g.ID, string(serialized))
	if err != nil {
		return err
	}
	return nil
}

func (gs *Tile38GameService) GameByID(gameID string) (*GameWithCoords, error) {
	f, err := gs.repo.FeatureByID("game", gameID)
	if err != nil {
		return nil, err
	}
	g, err := gs.getGame(gameID)
	if err != nil {
		return nil, err
	}
	return &GameWithCoords{Game: g, Coords: f.Coordinates}, nil
}

func (gs *Tile38GameService) getGame(gameID string) (*game.Game, error) {
	data, err := gs.repo.FeatureExtraData("game", gameID)
	if err != nil {
		return nil, err
	}
	g := &game.Game{}
	err = json.Unmarshal([]byte(data), g)
	return g, err
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
		g, err := gs.getGame(f.ID)
		if err != nil {
			log.Printf("[Tile38GameService] error to retrive game <%s> data", f.ID)
			continue
		}
		games[i] = GameWithCoords{
			Game:   g,
			Coords: f.Coordinates,
		}
	}

	return games, nil
}

func (gs *Tile38GameService) ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, action GamePlayerMove) error) error {
	return gs.stream.StreamIntersects(ctx, "player", "game", gameID, func(d *repository.Detection) error {
		p := model.Player{ID: d.FeatID, Lat: d.Lat, Lon: d.Lon}
		action := GamePlayerMoveInside
		if d.Intersects == repository.Exit {
			action = GamePlayerMoveOutside
		}
		return callback(p, action)
	})
}
