package dispatch

type Dispatchable interface {
	ShouldRun(state interface{}) bool
}
