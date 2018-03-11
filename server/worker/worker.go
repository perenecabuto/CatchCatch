package worker

// Manager for workers and its tasks
type Manager interface {
	Start()
	Stop()
	Add(w Worker)
	Run(w Worker, params map[string]interface{}) error
}

// Worker runs tasks
type Worker interface {
	ID() string
	Job(params map[string]interface{}) error
}
