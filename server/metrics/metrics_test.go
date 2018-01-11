package metrics

import (
	"math/rand"
	"testing"
)

func TestNotify(t *testing.T) {
	m, err := NewMetricsCollector("http://localhost:8086", "catchcatch", "", "")
	if err != nil {
		t.Skip(err)
	}
	if err := m.Ping(); err != nil {
		t.Skip(err)
	}

	tags := Tags{
		"host":   "localhost",
		"origin": "test",
	}
	values := Values{
		"connected-users": rand.Float64(),
	}
	if err := m.Notify("random", tags, values); err != nil {
		t.Fatal(err)
	}
}
