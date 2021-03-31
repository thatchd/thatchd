package utils

import (
	"context"
	"encoding/json"
	"fmt"

	thatchdv1alpha1 "github.com/thatchd/thatchd/api/v1alpha1"
	"github.com/thatchd/thatchd/pkg/thatchd/testsuite"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TestCaseState represents the curent state of a TestCase
type TestCaseState struct {
	Name        string                                `json:"name"`
	Annotations map[string]string                     `json:"annotations,omitempty"`
	Status      thatchdv1alpha1.TestCaseCurrentStatus `json:"status,omitempty"`
}

// TestCaseReconciler is a state reconciler for the TestCaseState. It reconciles
// the state from the current test cases in the namespace
type TestCaseReconciler struct{}

var _ testsuite.Reconciler = &TestCaseReconciler{}

func NewTestCaseReconciler() *TestCaseReconciler {
	return &TestCaseReconciler{}
}

func (r *TestCaseReconciler) ParseState(state string) (interface{}, error) {
	result := []TestCaseState{}
	if err := json.Unmarshal([]byte(state), &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (r *TestCaseReconciler) Reconcile(client k8sclient.Client, namespace string, currentState interface{}) (interface{}, error) {
	testCaseList := &thatchdv1alpha1.TestCaseList{}
	if err := client.List(context.TODO(), testCaseList, k8sclient.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("failed to list test cases: %w", err)
	}

	result := make([]TestCaseState, 0, len(testCaseList.Items))

	for _, testCase := range testCaseList.Items {
		result = append(result, TestCaseState{
			Name:        testCase.Name,
			Annotations: testCase.Annotations,
			Status:      testCase.Status.Status,
		})
	}

	return result, nil
}
