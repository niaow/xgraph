package xgraph

import "errors"

// Job is an operation in the execution graph
type Job interface {
	// Name returns the name of the Job.
	Name() string

	// The Run method runs the job.
	// Is called after the dependencies have been run successfully.
	Run() error

	// ShouldRun checks if the job should be run.
	// Job is marked as errored if this returns an error.
	// Dependents will be run even if this Job does not need to be run.
	ShouldRun() (bool, error)

	// Dependencies returns a list of dependencies for the Job.
	// If this returns an error, the Job is marked as errored.
	Dependencies() ([]string, error)
}

// BasicJob is a simple type which implements Job.
type BasicJob struct {
	// JobName of the BasicJob.
	// Required.
	JobName string

	// RunCallback is called when the BasicJob is run.
	// Required.
	RunCallback func() error

	// ShouldRunCallback returns whether the BasicJob should be run.
	// Defaults to a function that always returns true.
	ShouldRunCallback func() (bool, error)

	// Deps is a list of dependencies for the BasicJob.
	// Defaults to []string{}.
	Deps []string
}

// Name returns the name of the Job.
func (bj BasicJob) Name() string {
	return bj.JobName
}

// ErrMissingCallback indicates that a callback is missing on a BasicJob.
var ErrMissingCallback = errors.New("missing callback for BasicJob")

// Run runs the BasicJob, calling RunCallback.
// Returns ErrMissingCallback if RunCallback is nil.
func (bj BasicJob) Run() error {
	if bj.RunCallback == nil {
		return ErrMissingCallback
	}
	return bj.RunCallback()
}

// ShouldRun checks ifd the BasicJob should be run, using ShouldRunCallback.
// Returns ErrMissingCallback if ShouldRunCallback is nil.
func (bj BasicJob) ShouldRun() (bool, error) {
	if bj.ShouldRunCallback == nil {
		return true, nil
	}
	return bj.ShouldRunCallback()
}

// Dependencies returns the dependencies list of the BasicJob.
// Never returns an error.
// If Deps is nil, returns an empty slice for the dependencies.
func (bj BasicJob) Dependencies() ([]string, error) {
	if bj.Deps == nil {
		bj.Deps = []string{}
	}
	return bj.Deps, nil
}
