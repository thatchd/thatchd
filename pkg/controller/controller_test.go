package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	thatchdv1alpha1 "github.com/sergioifg94/thatchd/pkg/apis/thatchd/v1alpha1"
	testcasecontroller "github.com/sergioifg94/thatchd/pkg/controller/testcase"
	testprogramcontroller "github.com/sergioifg94/thatchd/pkg/controller/testprogram"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testcase"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testprogram"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type testScenario struct {
	Name                  string
	TestProgramReconciler testprogram.Reconciler
	TestCases             map[string]testcase.Interface
	TestCaseCRs           []*thatchdv1alpha1.TestCase
	TestProgramCR         *thatchdv1alpha1.TestProgram
	Assert                func(client client.Client, programReconcileResult reconcile.Result, programReconcileError error, testCaseResults map[string]*testCaseRun) error
}

type testCaseRun struct {
	Reconciler reconcile.Reconciler
	Result     reconcile.Result
	Error      error
}

type testProgramState struct {
	ComponentA componentStatus `json:"componentA"`
	ComponentB componentStatus `json:"componentB"`
}

type componentStatus struct {
	Ready   bool `json:"ready"`
	Healthy bool `json:"healthy"`
}

var scenario1 testScenario = testScenario{
	Name: "Tests for component 1 ready are dispatched",
	TestProgramCR: &thatchdv1alpha1.TestProgram{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-program",
			Namespace: "thatchd",
		},
		Spec: thatchdv1alpha1.TestProgramSpec{
			InitialState: "{}",
		},
	},
	TestCaseCRs: []*thatchdv1alpha1.TestCase{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-case-A",
				Namespace: "thatchd",
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-case-B",
				Namespace: "thatchd",
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:      "test-case-C",
				Namespace: "thatchd",
			},
			Spec: thatchdv1alpha1.TestCaseSpec{
				Timeout: addr("1s"),
			},
		},
	},
	TestProgramReconciler: &testProgramReconcilerMock{
		reconcile: func(client client.Client, currentState string) (interface{}, error) {
			return testProgramState{
				ComponentA: componentStatus{
					Ready:   true,
					Healthy: false,
				},
				ComponentB: componentStatus{
					Ready: false,
				},
			}, nil
		},
	},

	TestCases: map[string]testcase.Interface{
		"test-case-A": &testCaseInterfaceMock{
			shouldRun: func(testContext interface{}) bool {
				return testContext.(testProgramState).ComponentA.Ready
			},
			run: func(client client.Client) error {
				return errors.New("This test failed")
			},
		},
		"test-case-B": &testCaseInterfaceMock{
			shouldRun: func(testContext interface{}) bool {
				return testContext.(testProgramState).ComponentB.Ready
			},
			run: func(client client.Client) error {
				return errors.New("This test failed")
			},
		},
		"test-case-C": &testCaseInterfaceMock{
			shouldRun: func(testContext interface{}) bool {
				return true
			},
			run: func(client client.Client) error {
				time.Sleep(time.Second * 5)
				return nil
			},
		},
	},

	Assert: func(client client.Client, programReconcileResult reconcile.Result, programReconcileError error, testCaseResults map[string]*testCaseRun) error {
		testCaseACR := &thatchdv1alpha1.TestCase{}
		if err := client.Get(context.TODO(), types.NamespacedName{
			Name:      "test-case-A",
			Namespace: "thatchd",
		}, testCaseACR); err != nil {
			return fmt.Errorf("failed to retrieve test case A: %v", err)
		}

		if testCaseACR.Status.DispatchedAt == nil {
			return fmt.Errorf("expected test case A to be dispatched but it wasn't")
		}
		if *testCaseACR.Status.FailureMessage != "This test failed" {
			return fmt.Errorf("unexpected failure message. Got %s", *testCaseACR.Status.FailureMessage)
		}
		if testCaseACR.Status.FinishedAt == nil {
			return fmt.Errorf("expected test case A to be marked as finished, but wasn't")
		}
		if testCaseACR.Status.Status != thatchdv1alpha1.TestCaseFailed {
			return fmt.Errorf("expected test case A to be in failed status, but was %s", testCaseACR.Status.Status)
		}

		testCaseBCR := &thatchdv1alpha1.TestCase{}
		if err := client.Get(context.TODO(), types.NamespacedName{
			Name:      "test-case-B",
			Namespace: "thatchd",
		}, testCaseBCR); err != nil {
			return fmt.Errorf("failed to retrieve test case B: %v", err)
		}

		if testCaseBCR.Status.DispatchedAt != nil {
			return fmt.Errorf("expected test case B to not be dispatched")
		}

		testCaseCCR := &thatchdv1alpha1.TestCase{}
		if err := client.Get(context.TODO(), types.NamespacedName{
			Name:      "test-case-C",
			Namespace: "thatchd",
		}, testCaseCCR); err != nil {
			return fmt.Errorf("failed to retrieve test case C: %v", err)
		}

		if testCaseCCR.Status.FinishedAt == nil {
			return fmt.Errorf("expected test case C to be marked as finished but wasn't")
		}
		if testCaseCCR.Status.Status != thatchdv1alpha1.TestCaseCanceled {
			return fmt.Errorf("expected test case C to be marked as canceled, but was %s", testCaseCCR.Status.Status)
		}

		return nil
	},
}

var scenarios []testScenario = []testScenario{
	scenario1,
}

func TestThatchdControllers(t *testing.T) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Create scheme and client
			scheme := buildScheme(t)
			client := fake.NewFakeClientWithScheme(scheme, scenario.TestProgramCR)
			// Pre populate client with TestCase CRs
			for _, testCaseCR := range scenario.TestCaseCRs {
				if err := client.Create(context.TODO(), testCaseCR); err != nil {
					t.Fatalf("error pre-populating TestCase CR: %v", err)
				}
			}

			// Create program controller
			programcontroller := testprogramcontroller.NewReconciler(
				client,
				scheme,
				time.Minute,
				scenario.TestCases,
				scenario.TestProgramReconciler,
			)

			// Create test case run data
			testCases := map[string]*testCaseRun{}
			for testCaseName, testCaseInterface := range scenario.TestCases {
				testCases[testCaseName] = &testCaseRun{
					Reconciler: testcasecontroller.NewReconciler(
						client,
						scheme,
						testCaseName,
						testCaseInterface,
					),
				}
			}

			programReconcileResult, err := programcontroller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      scenario.TestProgramCR.Name,
				Namespace: scenario.TestProgramCR.Namespace,
			}})

			for _, testCaseCR := range scenario.TestCaseCRs {
				testCase := testCases[testCaseCR.Name]
				result, err := testCase.Reconciler.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      testCaseCR.Name,
					Namespace: testCaseCR.Namespace,
				}})

				testCase.Result = result
				testCase.Error = err
			}

			if err := scenario.Assert(client, programReconcileResult, err, testCases); err != nil {
				t.Error(err)
			}
		})
	}
}

func buildScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := thatchdv1alpha1.SchemeBuilder.AddToScheme(scheme); err != nil {
		t.Fatalf("error building scheme: %v", err)
	}

	return scheme
}

type testProgramReconcilerMock struct {
	reconcile func(client.Client, string) (interface{}, error)
}

var _ testprogram.Reconciler = &testProgramReconcilerMock{}

func (m *testProgramReconcilerMock) Reconcile(client client.Client, currentState string) (interface{}, error) {
	return m.reconcile(client, currentState)
}

type testCaseInterfaceMock struct {
	shouldRun func(testContext interface{}) bool
	run       func(client client.Client) error
}

var _ testcase.Interface = &testCaseInterfaceMock{}

func (m *testCaseInterfaceMock) ShouldRun(testContext interface{}) bool {
	return m.shouldRun(testContext)
}

func (m *testCaseInterfaceMock) Run(client client.Client) error {
	return m.run(client)
}

func addr(v string) *string {
	return &v
}
