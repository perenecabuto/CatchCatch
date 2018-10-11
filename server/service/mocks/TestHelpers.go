package mocks

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/stretchr/testify/assert"
)

// AssertPublished checks if dispacher has published a message on a topic
func AssertPublished(t *testing.T, d *Dispatcher, topic string, evt *core.GameEventPayload, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	received := []core.GameEventPayload{}

	for {
		select {
		case <-timer.C:
			t.Errorf("Nothing found or published on topic:%s\nexpected:%+v\nreceive:%+v", topic, evt, received)
			return false
		default:
			_t := &testing.T{}
			for _, c := range d.Calls {
				topicParam, data := c.Arguments.Get(0), c.Arguments.Get(1)
				if c.Method == "Publish" && topicParam.(string) == topic {
					ejson, _ := json.Marshal(evt)
					if assert.JSONEq(_t, string(ejson), string(data.([]byte))) {
						return true
					}
					p := core.GameEventPayload{}
					json.Unmarshal(data.([]byte), &p)
					received = append(received, p)
				}
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}
