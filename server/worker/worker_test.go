package worker_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/perenecabuto/CatchCatch/server/worker"
)

func TestJobLockName(t *testing.T) {
	cases := []struct {
		job      worker.Job
		expected string
	}{
		{worker.Job{},
			`catchcatch:worker:queue::{"params":null}:lock`},
		{worker.Job{ID: "1234"},
			`catchcatch:worker:queue::{"params":null}:lock`},
		{worker.Job{ID: "1234", Params: worker.TaskParams{"param1": "value1"}},
			`catchcatch:worker:queue::{"params":{"param1":"value1"}}:lock`},
		{worker.Job{ID: "1234", Params: worker.TaskParams{"param1": "value1", "param2": "value2"}},
			`catchcatch:worker:queue::{"params":{"param1":"value1","param2":"value2"}}:lock`},
		{worker.Job{ID: "1234", Params: worker.TaskParams{"param1": "value1", "param2": "value2"}, Unique: true},
			`catchcatch:worker:queue::{"params":{"param1":"value1","param2":"value2"}}:lock`},
		{worker.Job{ID: "1234", TaskID: "task1", Params: worker.TaskParams{"param1": "value1", "param2": "value2"}},
			`catchcatch:worker:queue:task1:{"params":{"param1":"value1","param2":"value2"}}:lock`},
		{worker.Job{ID: "1234", TaskID: "runner", Params: worker.TaskParams{"param1": "value1"}},
			`catchcatch:worker:queue:runner:{"params":{"param1":"value1"}}:lock`},
	}

	for _, tt := range cases {
		actual := tt.job.LockName()
		assert.Equal(t, tt.expected, actual)
	}
}

func TestIsLastUpdateBefore(t *testing.T) {
	now := time.Now()
	cases := []struct {
		job      worker.Job
		interval time.Duration
		expected bool
	}{
		{worker.Job{}, time.Hour, false},
		{worker.Job{LastUpdate: now}, time.Second, true},
		{worker.Job{LastUpdate: now}, time.Minute, true},
		{worker.Job{LastUpdate: now}, time.Hour, true},
		{worker.Job{LastUpdate: now.Add(-time.Second)}, time.Second * 2, true},
		{worker.Job{LastUpdate: now.Add(-time.Second * 2)}, time.Second, false},
		{worker.Job{LastUpdate: now.Add(-time.Hour * 2)}, time.Hour, false},
		{worker.Job{LastUpdate: now.Add(-(time.Hour + time.Minute))}, time.Hour, false},
		{worker.Job{LastUpdate: now.Add(-time.Minute * 2)}, time.Minute, false},
	}

	for _, tt := range cases {
		actual := tt.job.IsUpdatedToInterval(tt.interval)
		if !assert.Equal(t, tt.expected, actual) {
			t.Log(">", tt.job.LastUpdate, "should be before", now.Add(tt.interval))
		}
	}
}
