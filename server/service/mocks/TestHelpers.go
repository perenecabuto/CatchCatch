package mocks

import (
	"testing"
	"time"
)

// AssertPublished checks if dispacher has published a message on a topic
func AssertPublished(t *testing.T, m *Dispatcher, timeout time.Duration, topic string, assertCB func(data []byte) bool) bool {
	assert := func() bool {
		for _, c := range m.Calls {
			t, data := c.Arguments.Get(0), c.Arguments.Get(1)
			if c.Method == "Publish" && t.(string) == topic {
				if assertCB(data.([]byte)) {
					return true
				}
			}
		}
		return false
	}

	timer := time.NewTimer(timeout)
	for {
		select {
		case <-timer.C:
			t.Errorf("Nothing found or published on topic<%s>", topic)
			return false
		default:
			ok := assert()
			if ok {
				return ok
			}
			time.Sleep(time.Millisecond * 100)
		}
	}
}
