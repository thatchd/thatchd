package testcase

import (
	"fmt"

	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	ShouldRun(state interface{}) bool

	Run(client client.Client, namespace string) error
}

func FromStrategy(s *strategy.Strategy, providers []strategy.StrategyProvider) (Interface, error) {
	result := strategy.FromStrategy(s, providers)
	if result == nil {
		return nil, fmt.Errorf("no provider found for strategy %s", s)
	}

	typedResult, ok := result.(Interface)
	if !ok {
		return nil, fmt.Errorf("provider for strategy %s doesn't return testcase interface", s)
	}

	return typedResult, nil
}
