package main

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"

	sq "github.com/Masterminds/squirrel"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

type PlayerRankService struct {
	db sq.StatementBuilderType
}

func NewPlayerRankService(db *sql.DB) *PlayerRankService {
	dbCache := sq.NewStmtCacher(db)
	sb := sq.StatementBuilder.
		RunWith(dbCache).
		PlaceholderFormat(sq.Dollar)
	return &PlayerRankService{sb}
}

func (s *PlayerRankService) SetPlayer(ctx context.Context, p *Player) error {
	data := map[string]interface{}{}
	err := mapstructure.Decode(p, &data)
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = s.db.Insert("player").SetMap(data).ExecContext(ctx)
	return errors.Wrapf(err, "can't set player:%v", p)
}

func (s *PlayerRankService) GetPlayer(ctx context.Context, playerID string) (*Player, error) {
	p := &Player{}
	err := s.db.Select("ID", "Name", "PictureURL").
		From("player").
		Where("ID = ?", playerID).
		QueryRowContext(ctx).
		Scan(&p.ID, &p.Name, &p.PictureURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return p, nil
}

func (s *PlayerRankService) AddPlayerRank(ctx context.Context,
	gameName string, gameDate time.Time, playerID string, points int) (*Rank, error) {
	p, err := s.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if p == nil {
		p = &Player{ID: playerID}
		err := s.SetPlayer(ctx, p)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}
	gameDate = time.Unix(gameDate.UTC().Unix(), 0)
	r := &Rank{GameName: gameName, GameDate: gameDate, Player: p, Points: points}
	result, err := s.db.Insert("rank").SetMap(map[string]interface{}{
		"GameName": r.GameName, "GameDate": r.GameDate,
		"PlayerID": r.Player.ID, "Points": r.Points,
	}).ExecContext(ctx)
	if err != nil {
		return nil, errors.Cause(err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, errors.Cause(err)
	}
	if n != 1 {
		return nil, errors.New("no rows affected")
	}
	return r, nil
}
