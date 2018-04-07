package metrics_test

import (
	"math/rand"
	"testing"

	"github.com/perenecabuto/CatchCatch/server/metrics"
)

func TestNotify(t *testing.T) {
	m, err := metrics.NewCollector("http://localhost:8086", "catchcatch", "", "")
	if err != nil {
		t.Skip(err)
	}
	if err := m.Ping(); err != nil {
		t.Skip(err)
	}

	tags := metrics.Tags{
		"host":   "localhost",
		"origin": "test",
	}
	values := metrics.Values{
		"connected-users": rand.Float64(),
	}
	if err := m.Notify("random", tags, values); err != nil {
		t.Fatal(err)
	}
}
