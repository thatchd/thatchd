package strategy

import "reflect"

// +kubebuilder:object:generate=true
type Strategy struct {
	Provider      string            `json:"provider"`
	Configuration map[string]string `json:"configuration,omitempty"`
}

type StrategyProvider interface {
	New(configuration map[string]string) interface{}
}

func FromStrategy(strategy *Strategy, providers map[string]StrategyProvider) interface{} {
	provider, ok := providers[strategy.Provider]
	if !ok {
		return nil
	}

	return provider.New(strategy.Configuration)
}

// StrategyProviderFuncion is a Strategy provider implementation that delegates
// the strategy construction into a `new` function.
type StrategyProviderFunction struct {
	new func(map[string]string) interface{}
}

var _ StrategyProvider = &StrategyProviderFunction{}

func NewProviderFunction(new func(map[string]string) interface{}) StrategyProvider {
	return &StrategyProviderFunction{
		new: new,
	}
}

func (p *StrategyProviderFunction) New(configuration map[string]string) interface{} {
	return p.new(configuration)
}

// StrategyProviderForType is a StrategyProvider implementation that creates a
// strategy by creating an instance of the same type of the template object
type StrategyProviderForType struct {
	template interface{}
}

var _ StrategyProvider = &StrategyProviderForType{}

func NewProviderForType(template interface{}) StrategyProvider {
	return &StrategyProviderForType{
		template: template,
	}
}

func (p *StrategyProviderForType) New(_ map[string]string) interface{} {
	t := reflect.TypeOf(p.template)

	value := reflect.New(t)
	if t.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	return value.Interface()
}
