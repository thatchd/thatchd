package v1alpha1

import "github.com/sergioifg94/thatchd/pkg/thatchd/strategy"

// +kubebuilder:object:generate=true
type Strategy struct {
	strategy.Strategy `json:",inline"`
}
