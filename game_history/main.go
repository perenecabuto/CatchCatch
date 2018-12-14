package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo"
	nats "github.com/nats-io/go-nats"
	"github.com/pkg/errors"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/perenecabuto/CatchCatch/server/service/messages"
)

type PlayerHandler struct {
	service *PlayerRankService
}

var key = "catchcatch"

func (h *PlayerHandler) GetUser(c echo.Context) error {
	log.Println(c.Cookies())
	return nil
}
func (h *PlayerHandler) UserImage(c echo.Context) error {
	log.Println(c.Cookies())
	return nil
}

type RankHandler struct {
	service *PlayerRankService
}

func (h *RankHandler) ByUser(c echo.Context) error {
	log.Println(c.Cookies())
	return nil
}
func (h *RankHandler) ByGame(c echo.Context) error {
	log.Println(c.Cookies())
	return nil
}
func (h *RankHandler) Top(c echo.Context) error {

	return nil
}

func extractPlayerFromJWT(signed string, key interface{}) (*Player, error) {
	log.Println("parse jwt", signed, key)
	token, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !token.Valid {
		return nil, errors.New("token is invalid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("claims must be jwt.MapClains compatible")
	}
	id, ok := claims["sub"].(string)
	if !ok {
		id, ok = claims["jti"].(string)
	}
	if !ok {
		return nil, errors.New("id not present on claims")
	}
	name, _ := claims["name"].(string)
	pictureURL, _ := claims["picture"].(string)
	p := &Player{ID: id, Name: name, PictureURL: pictureURL}
	return p, nil
}

func main() {
	natsURL := nats.DefaultURL
	conn, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("nats", err.Error())
	}
	dispatcher := messages.NewNatsDispatcher(conn)
	worker := core.NewGameWorker(nil, dispatcher)

	const (
		host     = "localhost"
		port     = 5432
		user     = "catch"
		password = "catch"
		dbname   = "catchcatch"
	)

	psqlInfo := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("postgres", err.Error())
	}
	defer db.Close()

	service := NewPlayerRankService(db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn.Subscribe("user:logged", func(msg *nats.Msg) {
		signed := string(msg.Data)
		log.Println("token", signed)
		player, err := extractPlayerFromJWT(signed, []byte(key))
		log.Printf("extractPlayerFromJWT %+v %+v", player, err)
		err = service.SetPlayer(ctx, player)
		log.Printf("extractPlayerFromJWT postgres %+v", err)

	})

	err = worker.OnGameEvent(ctx, func(payload *core.GameEventPayload) error {
		if payload.Event == core.GameFinished {
			gameName, gameDate := payload.Game, time.Now()
			log.Println("rank!!!", payload.Rank.PlayerRank)
			for _, r := range payload.Rank.PlayerRank {
				rank, err := service.AddPlayerRank(ctx, gameName, gameDate, r.Player.ID, r.Points)
				if err != nil {
					return err
				}
				log.Println("added rank:", rank)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal("can't listen to game events")
	}

	rankH := &RankHandler{service}
	playerH := &PlayerHandler{service}

	e := echo.New()
	e.GET("/user/:id", playerH.GetUser)
	e.GET("/user/:id/image", playerH.UserImage)
	e.GET("/rank/by-user/:id", rankH.ByUser)
	e.GET("/rank/by-game/:id", rankH.ByGame)
	e.GET("/rank/top/:limit/:page", rankH.Top)

	e.Logger.Fatal(e.Start(":8888"))
}
