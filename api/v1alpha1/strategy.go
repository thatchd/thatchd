package v1alpha1

import (
	"github.com/thatchd/thatchd/pkg/thatchd/strategy"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:generate=true
type Strategy struct {
	strategy.Strategy `json:",inline"`
}

// +kubebuilder:object:generate=false
type StrategyBacked interface {
	runtime.Object

	GetStrategy() Strategy
}
