package testworker

import (
	"context"
	"fmt"

	"github.com/thatchd/thatchd/pkg/thatchd/dispatch"
	"github.com/thatchd/thatchd/pkg/thatchd/strategy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MutateStateFn func(interface{}) (interface{}, error)

type Interface interface {
	dispatch.Dispatchable

	Run(ctx context.Context, namespace string, client client.Client) (MutateStateFn, error)
}

func FromStrategy(s *strategy.Strategy, providers map[string]strategy.StrategyProvider) (Interface, error) {
	result := strategy.FromStrategy(s, providers)
	if result == nil {
		return nil, fmt.Errorf("no provider found for strategy %s", s)
	}

	typedResult, ok := result.(Interface)
	if !ok {
		return nil, fmt.Errorf("provider for strategy %s doesn't return testworker interface", s)
	}

	return typedResult, nil
}

// NoMutate is used when the test worker doesn't mutate the
// test state
func NoMutate(state interface{}) (interface{}, error) {
	return state, nil
}
