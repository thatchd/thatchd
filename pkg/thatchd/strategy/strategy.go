package strategy

import (
	"reflect"
)

// +k8s:deepcopy-gen
type Strategy struct {
	Provider      string            `json:"provider"`
	Configuration map[string]string `json:"configuration,omitempty"`
}

type StrategyProvider interface {
	New(configuration map[string]string) interface{}
}

func FromStrategy(strategy *Strategy, providers []StrategyProvider) interface{} {
	for _, provider := range providers {
		value := reflect.ValueOf(provider)
		name := value.Type().Elem().Name()
		if name == strategy.Provider {
			return provider.New(strategy.Configuration)
		}
	}

	return nil
}
