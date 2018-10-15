package mocks

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/perenecabuto/CatchCatch/server/core"
	"github.com/stretchr/testify/assert"
	funk "github.com/thoas/go-funk"
)

// AssertPublished checks if dispacher has published a message on a topic
func AssertPublished(t *testing.T, d *Dispatcher, topic string, evt *core.GameEventPayload, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	received := []core.GameEventPayload{}

	for {
		select {
		case <-timer.C:
			events := funk.Reverse(received).([]core.GameEventPayload)[:5]
			eventsStr := funk.Map(events, func(e core.GameEventPayload) string {
				return fmt.Sprint(e)
			}).([]string)
			t.Errorf(
				"Nothing found or published on topic:%v\nExpected:\n%v\nReceived:\n%s",
				topic, *evt, strings.Join(eventsStr, "\n------\n"),
			)
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
