package example

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testsuite"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodStatus string

var PodReady PodStatus = "Ready"
var PodNotReady PodStatus = "NotReady"
var PodTested PodStatus = "Tested"
var PodAnnotated PodStatus = "Annotated"

// PodSuiteState represent the current testing state as an association between
// pod names -> whether they're ready or not
type PodSuiteState map[string]PodStatus

// PodsSuiteReconciler reconciles the testing state with the pods in the
// namespace
type PodsSuiteReconciler struct{}

var _ testsuite.Reconciler = &PodsSuiteReconciler{}

func (r *PodsSuiteReconciler) ParseState(state string) (interface{}, error) {
	result := PodSuiteState{}
	err := json.Unmarshal([]byte(state), &result)
	return result, err
}

func (r *PodsSuiteReconciler) Reconcile(c client.Client, namespace string, s interface{}) (interface{}, error) {
	currentState := s.(PodSuiteState)

	podList := &corev1.PodList{}
	if err := c.List(context.TODO(), podList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %v", namespace, err)
	}

	for _, pod := range podList.Items {
		podState, ok := currentState[pod.Name]
		if ok && (podState == PodTested || podState == PodAnnotated) {
			continue
		}

		podState = PodNotReady
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodRunning {
			podState = PodReady
		}

		currentState[pod.Name] = podState
	}

	return currentState, nil
}

func NewPodsSuiteProvider() strategy.StrategyProvider {
	return strategy.NewProviderForType(&PodsSuiteReconciler{})
}
