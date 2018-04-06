package mocks

import "testing"

// AssertPublished checks if dispacher has published a message on a topic
func AssertPublished(t *testing.T, m *Dispatcher, topic string, assertCB func(data []byte) bool) bool {
	for _, c := range m.Calls {
		t, data := c.Arguments.Get(0), c.Arguments.Get(1)
		if c.Method == "Publish" && t.(string) == topic {
			if assertCB(data.([]byte)) {
				return true
			}
		}
	}
	t.Errorf("Nothing found or published on topic<%s>", topic)
	return false
}
