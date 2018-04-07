package worker

import (
	"context"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/metrics"
)

const (
	RunMetricsName   = "catchcatch.worker.run.metrics"
	StartMetricsName = "catchcatch.worker.start.metrics"
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
	tags := metrics.Tags{"host": w.options.Host, "origin": w.options.Origin, "id": w.ID()}
	values := metrics.Values{"params": params}

	if err := w.collector.Notify(StartMetricsName, tags, values); err != nil {
		log.Println("[WorkerWithMetrics] metrics error:", err)
	}

	start := time.Now()
	err := w.Worker.Run(ctx, params)
	values["elapsed"] = int(time.Since(start) / time.Millisecond)
	if err != nil {
		values["error"] = err.Error()
	}
	log.Println("[WorkerWithMetrics]", RunMetricsName, tags, values)
	if err := w.collector.Notify(RunMetricsName, tags, values); err != nil {
		log.Println("[WorkerWithMetrics] metrics error:", err)
	}
	return nil
}
