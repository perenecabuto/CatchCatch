package game

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// MarshalJSON implemente json.Marsheler
func (g *Game) MarshalJSON() ([]byte, error) {
	data, _ := sjson.SetBytes([]byte{}, "id", g.ID)
	data, _ = sjson.SetBytes(data, "started", g.started)
	data, _ = sjson.SetBytes(data, "targetID", g.TargetID())
	data, _ = sjson.SetBytes(data, "players", g.players.AsMap())
	return data, nil
}

// UnmarshalJSON implemente json.Unmarsheler
func (g *Game) UnmarshalJSON(data []byte) error {
	g.ID = gjson.GetBytes(data, "id").String()
	g.started = int32(gjson.GetBytes(data, "started").Int())
	g.targetID.Store(gjson.GetBytes(data, "targetID").String())
	g.players = NewGamePlayers()
	pdata := gjson.GetBytes(data, "players").Map()
	for _, v := range pdata {
		p := Player{}
		if err := json.Unmarshal([]byte(v.Raw), &p); err != nil {
			return err
		}
		g.players.Set(p)
	}
	return nil
}