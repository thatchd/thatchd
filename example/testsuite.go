package example

import (
	"context"
	"fmt"

	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testsuite"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PodSuiteState represent the current testing state as an association between
// pod names -> whether they're ready or not
type PodSuiteState map[string]bool

// PodsSuiteReconciler reconciles the testing state with the pods in the
// namespace
type PodsSuiteReconciler struct{}

var _ testsuite.Reconciler = &PodsSuiteReconciler{}

func (r *PodsSuiteReconciler) Reconcile(c client.Client, namespace, currentState string) (interface{}, error) {
	podList := &corev1.PodList{}
	if err := c.List(context.TODO(), podList, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list pods in namespace %s: %v", namespace, err)
	}

	result := PodSuiteState{}
	for _, pod := range podList.Items {
		result[pod.Name] = pod.Status.Phase == corev1.PodSucceeded ||
			pod.Status.Phase == corev1.PodRunning
	}

	return result, nil
}

type PodsSuiteProvider struct{}

var _ strategy.StrategyProvider = &PodsSuiteProvider{}

func (p *PodsSuiteProvider) New(_ map[string]string) interface{} {
	return &PodsSuiteReconciler{}
}
