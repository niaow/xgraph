package xgraph

// EventHandler is an interface used to process build events
type EventHandler interface {
	// OnQueued is called when a Job has been queued (waiting for dependencies)
	OnQueued(job string)

	// OnStart is called when a Job has been started
	OnStart(job string)

	// OnFinish is called when a Job has finished
	OnFinish(job string)

	// OnError is called when a Job fails
	OnError(job string, err error)
}

type nophandler struct{}

func (n nophandler) OnQueued(job string)           {}
func (n nophandler) OnStart(job string)            {}
func (n nophandler) OnFinish(job string)           {}
func (n nophandler) OnError(job string, err error) {}

// NoOpEventHandler is an EventHandler which does nothing
var NoOpEventHandler EventHandler = nophandler{}
