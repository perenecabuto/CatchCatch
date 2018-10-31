package worker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/perenecabuto/CatchCatch/server/worker"
)

func TestTaskLockName(t *testing.T) {
	cases := []struct {
		task     worker.Task
		expected string
	}{
		{worker.Task{},
			`catchcatch:worker:queue::{"params":null}:lock`},
		{worker.Task{ID: "1234"},
			`catchcatch:worker:queue::{"params":null}:lock`},
		{worker.Task{ID: "1234", Params: worker.TaskParams{"param1": "value1"}},
			`catchcatch:worker:queue::{"params":{"param1":"value1"}}:lock`},
		{worker.Task{ID: "1234", Params: worker.TaskParams{"param1": "value1", "param2": "value2"}},
			`catchcatch:worker:queue::{"params":{"param1":"value1","param2":"value2"}}:lock`},
		{worker.Task{ID: "1234", Params: worker.TaskParams{"param1": "value1", "param2": "value2"}, Unique: true},
			`catchcatch:worker:queue::{"params":{"param1":"value1","param2":"value2"}}:lock`},
		{worker.Task{ID: "1234", WorkerID: "worker1", Params: worker.TaskParams{"param1": "value1", "param2": "value2"}},
			`catchcatch:worker:queue:worker1:{"params":{"param1":"value1","param2":"value2"}}:lock`},
		{worker.Task{ID: "1234", WorkerID: "runner", Params: worker.TaskParams{"param1": "value1"}},
			`catchcatch:worker:queue:runner:{"params":{"param1":"value1"}}:lock`},
	}

	for _, tt := range cases {
		actual := tt.task.LockName()
		assert.Equal(t, tt.expected, actual)
	}
}
