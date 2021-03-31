package utils

import (
	"reflect"
	"testing"

	thatchdv1alpha1 "github.com/thatchd/thatchd/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTestCaseReconciler(t *testing.T) {
	scenarios := []struct {
		Name          string
		TestCases     []runtime.Object
		CurrentState  []TestCaseState
		ExpectedState []TestCaseState
	}{
		{
			Name: "Test cases reconciled",
			TestCases: []runtime.Object{
				&thatchdv1alpha1.TestCase{
					ObjectMeta: v1.ObjectMeta{
						Name:      "test-case1",
						Namespace: "thatchd",
						Annotations: map[string]string{
							"test-component": "component1",
						},
					},
					Status: thatchdv1alpha1.TestCaseStatus{
						Status: thatchdv1alpha1.TestCaseDispatched,
					},
				},
			},
			CurrentState: []TestCaseState{},
			ExpectedState: []TestCaseState{
				{
					Name: "test-case1",
					Annotations: map[string]string{
						"test-component": "component1",
					},
					Status: thatchdv1alpha1.TestCaseDispatched,
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			thatchdv1alpha1.AddToScheme(scheme)
			client := fake.NewFakeClientWithScheme(scheme, scenario.TestCases...)

			reconciler := NewTestCaseReconciler()
			state, err := reconciler.Reconcile(client, "thatchd", scenario.CurrentState)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(state, scenario.ExpectedState) {
				t.Errorf("unexpected reconciled state value. Expected %v, but got %v", scenario.ExpectedState, state)
			}
		})
	}
}
