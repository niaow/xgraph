package xgraph

//Executor is a tool which runs jobs on a Graph.
//It will complete as many jobs as possible and then return an error set if applicable.
type Executor struct {
	graph  *Graph
	runner WorkRunner
}

// JobStatus is the status of a Job.
type JobStatus string

// JobStatus constants
const (
	StatusPreparing JobStatus = "preparing"
	StatusQueued    JobStatus = "queued"
	StatusRunning   JobStatus = "running"
	StatusComplete  JobStatus = "complete"
	StatusFailed    JobStatus = "failed"
)

// StatusHandler is a type that handles status update events
type StatusHandler interface {
	OnStatusUpdate(name string, err error)
}
