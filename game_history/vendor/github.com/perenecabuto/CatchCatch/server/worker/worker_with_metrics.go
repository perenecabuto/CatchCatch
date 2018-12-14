package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/perenecabuto/CatchCatch/server/metrics"
)

const (
	RunMetricsName   = "catchcatch.worker.run.metrics"
	StartMetricsName = "catchcatch.worker.start.metrics"
)

type TaskWithMetrics struct {
	Task
	collector metrics.Collector
	options   MetricsOptions
}

type MetricsOptions struct {
	Host   string
	Origin string
	Params []string
}

func NewTaskWithMetrics(w Task, m metrics.Collector, opt MetricsOptions) Task {
	return &TaskWithMetrics{w, m, opt}
}

func (w TaskWithMetrics) Run(ctx context.Context, params TaskParams) error {
	var paramsValue string
	for _, p := range w.options.Params {
		if v, ok := params[p]; ok {
			paramsValue += fmt.Sprintf("%s=%v,", p, v)
		}
	}
	tags := metrics.Tags{
		"host": w.options.Host, "origin": w.options.Origin, "id": w.ID(),
		"params": paramsValue,
	}
	values := metrics.Values{"params": params}

	if err := w.collector.Notify(StartMetricsName, tags, values); err != nil {
		log.Println("[TaskWithMetrics] metrics error:", err)
	}

	start := time.Now()
	err := w.Task.Run(ctx, params)
	values["elapsed"] = int(time.Since(start) / time.Millisecond)
	if err != nil {
		values["error"] = err.Error()
	}
	log.Println("[TaskWithMetrics]", RunMetricsName, tags, values)
	if err := w.collector.Notify(RunMetricsName, tags, values); err != nil {
		log.Println("[TaskWithMetrics] metrics error:", err)
	}
	return nil
}
