package strategy

import "testing"

type mockProviderA struct{}
type mockProviderB struct{}

var providerA StrategyProvider = &mockProviderA{}
var providerB StrategyProvider = &mockProviderB{}

func TestNewTestCaseScheduleStrategy(t *testing.T) {

	providers := []StrategyProvider{
		&mockProviderA{},
		&mockProviderB{},
	}

	strategyA := FromStrategy(&Strategy{
		Provider:      "mockProviderA",
		Configuration: make(map[string]string),
	}, providers)

	if strategyA.(string) != "A" {
		t.Errorf("Expected strategyA to be \"A\", but got %v", strategyA)
	}

	strategyB := FromStrategy(&Strategy{
		Provider:      "mockProviderB",
		Configuration: make(map[string]string),
	}, providers)

	if strategyB.(string) != "B" {
		t.Errorf("Expected strategyB to be \"B\", but got %v", strategyB)
	}
}

func (a *mockProviderA) New(configuration map[string]string) interface{} {
	return "A"
}

func (b *mockProviderB) New(configuration map[string]string) interface{} {
	return "B"
}
