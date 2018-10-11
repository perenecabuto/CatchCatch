package worker_test

import (
	"context"
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

func TestWorkerWithMetricsSendRunMetrics(t *testing.T) {
	m := &mmocks.Collector{}
	w := &wmocks.Worker{}
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
	m.AssertCalled(t, "Notify", worker.RunMetricsName,
		mock.MatchedBy(func(actual metrics.Tags) bool {
			expected := metrics.Tags{"id": id, "host": opts.Host, "origin": opts.Origin}
			return assert.Equal(t, expected, actual)
		}),
		mock.MatchedBy(func(actual metrics.Values) bool {
			expected := metrics.Values{
				"params":  params,
				"elapsed": 100,
				"error":   err.Error(),
			}
			return assert.Equal(t, expected, actual)
		}),
	)
}

func TestWorkerWithMetricsSendStartMetrics(t *testing.T) {
	m := &mmocks.Collector{}
	w := &wmocks.Worker{}
	opts := worker.MetricsOptions{Host: "testhost", Origin: "test"}
	wm := worker.NewWorkerWithMetrics(w, m, opts)

	ctx := context.Background()
	err := errors.New("MockWorkerError")
	id := "MockWorker"
	w.On("ID").Return(id)
	w.On("Run", ctx, mock.Anything).Return(err)
	m.On("Notify", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := worker.TaskParams{"key1": "val1"}
	wm.Run(ctx, params)

	m.AssertCalled(t, "Notify", worker.StartMetricsName,
		metrics.Tags{"id": id, "host": opts.Host, "origin": opts.Origin},
		mock.Anything)
}
