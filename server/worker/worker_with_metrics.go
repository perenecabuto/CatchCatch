package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/metrics"
)

const (
	MetricsName = "catchcatch:worker:metrics"
)

type WorkerWithMetrics struct {
	Worker
	collector metrics.Collector
	options   MetricsOptions
}

type MetricsOptions struct {
	Host   string
	Origin string
}

func NewWorkerWithMetrics(w Worker, m metrics.Collector, opt MetricsOptions) Worker {
	return &WorkerWithMetrics{w, m, opt}
}

func (w WorkerWithMetrics) Run(ctx context.Context, params TaskParams) error {
	start := time.Now()
	err := w.Worker.Run(ctx, params)
	elapsed := time.Since(start)

	tags := metrics.Tags{
		"host":   w.options.Host,
		"origin": w.options.Origin,
	}
	data, _ := json.Marshal(params)
	values := metrics.Values{
		"elapsed": elapsed,
		"err":     err.Error(),
		"id":      w.ID(),
		"params":  data,
	}
	if err := w.collector.Notify(MetricsName, tags, values); err != nil {
		log.Println("[WorkerWithMetrics] metrics error:", err)
	}
	return err
}
