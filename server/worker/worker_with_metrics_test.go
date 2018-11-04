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

func TestTaskWithMetricsSendRunMetrics(t *testing.T) {
	m := &mmocks.Collector{}
	w := &wmocks.Task{}
	opts := worker.MetricsOptions{
		Host:   "testhost",
		Origin: "test",
		Params: []string{"key1", "key3"},
	}
	wm := worker.NewTaskWithMetrics(w, m, opts)

	ctx := context.Background()
	err := errors.New("MockTaskError")
	id := "MockTask"
	w.On("ID").Return(id)
	w.On("Run", ctx, mock.MatchedBy(func(worker.TaskParams) bool {
		time.Sleep(time.Millisecond * 100)
		return true
	})).Return(err)

	m.On("Notify", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := worker.TaskParams{"key1": "val1", "key2": "val2", "key3": "val3"}
	wm.Run(ctx, params)

	w.AssertCalled(t, "Run", ctx, mock.MatchedBy(func(actual worker.TaskParams) bool {
		return assert.Equal(t, params, actual)
	}))
	m.AssertCalled(t, "Notify", worker.RunMetricsName,
		mock.MatchedBy(func(actual metrics.Tags) bool {
			expected := metrics.Tags{"id": id, "host": opts.Host,
				"origin": opts.Origin, "params": "key1=val1,key3=val3,"}
			return assert.Equal(t, expected, actual)
		}),
		mock.MatchedBy(func(actual metrics.Values) bool {
			return assert.Equal(t, params, actual["params"]) &&
				assert.Equal(t, 100, actual["elapsed"]) &&
				assert.Contains(t, err.Error(), actual["error"])
		}),
	)
}

func TestTaskWithMetricsSendStartMetrics(t *testing.T) {
	m := &mmocks.Collector{}
	w := &wmocks.Task{}
	opts := worker.MetricsOptions{Host: "testhost", Origin: "test", Params: []string{"param2"}}
	wm := worker.NewTaskWithMetrics(w, m, opts)

	ctx := context.Background()
	err := errors.New("MockTaskError")
	id := "MockTask"
	w.On("ID").Return(id)
	w.On("Run", ctx, mock.Anything).Return(err)
	m.On("Notify", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	params := worker.TaskParams{"key1": "val1", "param2": "val-param2"}
	wm.Run(ctx, params)

	m.AssertCalled(t, "Notify", worker.StartMetricsName,
		metrics.Tags{"id": id, "host": opts.Host, "origin": opts.Origin, "params": "param2=val-param2,"},
		mock.Anything)
}
