package service

import (
	"context"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/game"
	"github.com/perenecabuto/CatchCatch/server/model"
	"github.com/perenecabuto/CatchCatch/server/service/repository"
	gjson "github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	// DefaultGeoEventRange set the watcher radar radius size
	DefaultGeoEventRange = 5000
)

type GameService interface {
	Create(gameID, serverID string) error
	Update(g *game.Game, serverID string, evt game.Event) error
	Remove(gameID string) error
	IsGameRunning(gameID string) (bool, error)
	GameByID(gameID string) (*game.Game, *game.Event, error)

	ObserveGamePlayers(ctx context.Context, gameID string, callback func(p model.Player, exit bool) error) error
	ObservePlayersCrossGeofences(ctx context.Context, callback func(string, model.Player) error) error
	ObserveGamesEvents(ctx context.Context, callback func(*game.Game, *game.Event) error) error
}

type Tile38GameService struct {
	repo   repository.Repository
	stream repository.EventStream
}

func NewGameService(repo repository.Repository, stream repository.EventStream) GameService {
	return &Tile38GameService{repo, stream}
}

func (gs *Tile38GameService) Create(gameID string, serverID string) error {
	f, err := gs.repo.FeatureByID("geofences", gameID)
	if err != nil {
		return err
	}
	_, err = gs.repo.SetFeature("game", gameID, f.Coordinates)
	if err != nil {
		return err
	}

	extra, _ := sjson.Set("", "updated_at", time.Now().Unix())
	extra, _ = sjson.Set(extra, "server_id", serverID)
	return gs.repo.SetFeatureExtraData("game", gameID, extra)
}

func (gs *Tile38GameService) IsGameRunning(gameID string) (bool, error) {
	data, err := gs.repo.FeatureExtraData("game", gameID)
	if err == repository.ErrFeatureNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	lastUpdate := time.Unix(gjson.Get(data, "updated_at").Int(), 0)
	expiration := lastUpdate.Add(20 * time.Second)
	return time.Now().Before(expiration), nil
}

func (gs *Tile38GameService) Update(g *game.Game, serverID string, evt game.Event) error {
	extra, _ := sjson.Set("", "event", evt)
	extra, _ = sjson.Set(extra, "updated_at", time.Now().Unix())
	extra, _ = sjson.Set(extra, "server_id", serverID)
	extra, _ = sjson.Set(extra, "players", g.Players())
	extra, _ = sjson.Set(extra, "started", g.Started())
	return gs.repo.SetFeatureExtraData("game", g.ID, extra)
}

func (gs *Tile38GameService) GameByID(gameID string) (*game.Game, *game.Event, error) {
	data, err := gs.repo.FeatureExtraData("game", gameID)
	if err == repository.ErrFeatureNotFound {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	players := make(map[string]*game.Player)
	pdata := gjson.Get(data, "players").Array()
	for _, pd := range pdata {
		p := &game.Player{
			Player:       model.Player{ID: pd.Get("id").String(), Lon: pd.Get("lon").Float(), Lat: pd.Get("lat").Float()},
			Role:         game.Role(pd.Get("Role").String()),
			DistToTarget: pd.Get("DistToTarget").Float(),
			Loose:        pd.Get("Loose").Bool(),
		}
		players[p.ID] = p
	}

	edata := gjson.Get(data, "event")
	evt := &game.Event{Name: game.EventName(edata.Get("Name").String())}
	if p := players[edata.Get("Player.ID").String()]; p != nil {
		evt.Player = *p
	}

	started := gjson.Get(data, "started").Bool()
	var targetID string
	for _, p := range players {
		if p.Role == game.GameRoleTarget {
			targetID = p.ID
			break
		}
	}

	return game.NewGameWithParams(gameID, started, players, targetID), evt, nil
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

func (gs *Tile38GameService) ObserveGamesEvents(ctx context.Context, callback func(*game.Game, *game.Event) error) error {
	return gs.stream.StreamNearByEvents(ctx, "game", "player", "*", DefaultGeoEventRange, func(d *repository.Detection) error {
		gameID, playerID := d.FeatID, d.NearByFeatID
		game, evt, err := gs.GameByID(gameID)
		if err != nil {
			return err
		}
		log.Println("game:event", evt, ":game:", game, ":player:", playerID)
		if game == nil {
			return nil
		}

		return callback(game, evt)
	})
}
