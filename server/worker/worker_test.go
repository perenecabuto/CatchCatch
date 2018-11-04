package worker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/perenecabuto/CatchCatch/server/worker"
)

func TestTaskLockName(t *testing.T) {
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
