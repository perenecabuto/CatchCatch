package model

import (
	"fmt"

	geo "github.com/kellydunn/golang-geo"
)

// Feature wraps geofence name and its geojeson
type Feature struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Coordinates string `json:"coords"`
}

// Player payload
type Player struct {
	ID  string  `json:"id"`
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}

func (p Player) String() string {
	return fmt.Sprintln("id:", p.ID, "lat:", p.Lat, "lon:", p.Lon)
}

// PlayerList list is an alias to []*Player
type PlayerList []*Player

// Point returns geo.Point with coordinates
func (p Player) Point() *geo.Point {
	return geo.NewPoint(p.Lat, p.Lon)
}

// DistTo returns the distance to other player
func (p Player) DistTo(other Player) float64 {
	return p.Point().GreatCircleDistance(other.Point()) * 1000
}
