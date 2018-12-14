package main

import "time"

type Player struct {
	ID         string
	Name       string
	PictureURL string
}

type Rank struct {
	GameName string
	GameDate time.Time
	Player   *Player
	Points   int
}
