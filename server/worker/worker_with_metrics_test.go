package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/perenecabuto/CatchCatch/server/metrics"
	"github.com/perenecabuto/CatchCatch/server/worker"

	mmocks "github.com/perenecabuto/CatchCatch/server/metrics/mocks"
	wmocks "github.com/perenecabuto/CatchCatch/server/worker/mocks"
)

func TestWorkerWithMetricsSendWhenTaskStarted(t *testing.T) {
	m := new(mmocks.Collector)
	w := new(wmocks.Worker)
	opts := worker.MetricsOptions{
		Host:   "testhost",
		Origin: "test",
	}
	wm := worker.NewWorkerWithMetrics(w, m, opts)

	ctx := context.Background()
	err := errors.New("MockWorkerError")
	id := "MockWorker"
	w.On("ID").Return(id)
	w.On("Run", ctx, mock.MatchedBy(func(worker.TaskParams) bool {
		time.Sleep(time.Millisecond * 100)
		return true
	})).Return(err)

	m.On("Notify", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := worker.TaskParams{"key1": "val1"}
	wm.Run(ctx, params)

	w.AssertCalled(t, "Run", ctx, mock.MatchedBy(func(actual worker.TaskParams) bool {
		return assert.Equal(t, params, actual)
	}))
	m.AssertCalled(t, "Notify", worker.MetricsName,
		mock.MatchedBy(func(actual metrics.Tags) bool {
			expected := metrics.Tags{"host": opts.Host, "origin": opts.Origin}
			return assert.Equal(t, expected, actual)
		}),
		mock.MatchedBy(func(actual metrics.Values) bool {
			data, _ := json.Marshal(params)
			expected := metrics.Values{
				"id":      id,
				"params":  data,
				"elapsed": 100,
				"err":     err.Error(),
			}
			elapsed := actual["elapsed"].(time.Duration)
			actual["elapsed"] = int(elapsed / time.Millisecond)
			return assert.Equal(t, expected, actual)
		}),
	)
}
