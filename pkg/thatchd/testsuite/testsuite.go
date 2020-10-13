package testsuite

import (
	"fmt"

	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler interface {
	ParseState(state string) (interface{}, error)

	Reconcile(client client.Client, namespace string, currentState interface{}) (interface{}, error)
}

func FromStrategy(s *strategy.Strategy, providers []strategy.StrategyProvider) (Reconciler, error) {
	result := strategy.FromStrategy(s, providers)
	if result == nil {
		return nil, fmt.Errorf("no provider found for strategy %s", s)
	}

	typedResult, ok := result.(Reconciler)
	if !ok {
		return nil, fmt.Errorf("provider for strategy %s doesn't return testprogram reconciler", s)
	}

	return typedResult, nil
}
