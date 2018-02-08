package game

import (
	"encoding/json"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// MarshalJSON implemente json.Marsheler
func (g Game) MarshalJSON() ([]byte, error) {
	data, _ := sjson.SetBytes([]byte{}, "id", g.ID)
	data, _ = sjson.SetBytes(data, "started", g.started)
	data, _ = sjson.SetBytes(data, "targetID", g.targetID)
	data, _ = sjson.SetBytes(data, "players", g.Players())
	return data, nil
}

// UnmarshalJSON implemente json.Unmarsheler
func (g *Game) UnmarshalJSON(data []byte) error {
	g.ID = gjson.GetBytes(data, "id").String()
	g.started = gjson.GetBytes(data, "started").Bool()
	g.targetID = gjson.GetBytes(data, "targetID").String()
	pdata := gjson.GetBytes(data, "players").Array()
	g.players = map[string]*Player{}
	for _, v := range pdata {
		p := Player{}
		if err := json.Unmarshal([]byte(v.Raw), &p); err != nil {
			return err
		}
		g.players[p.ID] = &p
	}
	return nil
}
