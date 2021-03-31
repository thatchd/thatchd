/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testingv1alpha1 "github.com/thatchd/thatchd/api/v1alpha1"
	"github.com/thatchd/thatchd/pkg/thatchd/strategy"
	"github.com/thatchd/thatchd/pkg/thatchd/testsuite"
	"github.com/thatchd/thatchd/pkg/thatchd/testworker"
)

// TestWorkerReconciler reconciles a TestWorker object
type TestWorkerReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	StrategyProviders map[string]strategy.StrategyProvider
}

// +kubebuilder:rbac:groups=testing.thatchd.io,resources=testworkers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=testing.thatchd.io,resources=testworkers/status,verbs=get;update;patch

func (r *TestWorkerReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("testworker", req.NamespacedName)

	instance := &testingv1alpha1.TestWorker{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		// Request object not found, could have been deleted after reconcile request.
		// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
		// Return and don't requeue
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Test worker hasn't been dispatched
	if instance.Status.DispatchedAt == nil {
		return ctrl.Result{}, nil
	}

	// Test worker has already started
	if instance.Status.StartedAt != nil {
		return ctrl.Result{}, err
	}

	// Set the StartedAt field
	instance.Status.StartedAt = testingv1alpha1.TimeString(time.Now())
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Run the test. If it failed, set the failure message and finish
	mutateState, err := r.runTest(ctx, instance)
	if err != nil {
		failureMessage := err.Error()
		instance.Status.FailureMessage = &failureMessage

		return ctrl.Result{}, r.Status().Update(ctx, instance)
	}

	// Update the test suite with the resulting mutating function
	if err := r.updateSuiteState(ctx, instance, mutateState); err != nil {
		return ctrl.Result{}, err
	}

	// Set the FinishedAt field
	instance.Status.FinishedAt = testingv1alpha1.TimeString(time.Now())
	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *TestWorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testingv1alpha1.TestWorker{}).
		Complete(r)
}

func (r *TestWorkerReconciler) runTest(ctx context.Context, instance *testingv1alpha1.TestWorker) (testworker.MutateStateFn, error) {
	str := strategy.Strategy(instance.Spec.Strategy.Strategy)

	testWorkerInterface, err := testworker.FromStrategy(&str, r.StrategyProviders)
	if err != nil {
		return nil, fmt.Errorf("error obtaining strategy for test worker %s: %v", instance.Name, err)
	}

	mutateState, err := testWorkerInterface.Run(ctx, instance.Namespace, r)
	if err != nil {
		testError := err.Error()
		instance.Status.FailureMessage = &testError

		return nil, r.Status().Update(ctx, instance)
	}

	return mutateState, nil
}

func (r *TestWorkerReconciler) updateSuiteState(ctx context.Context, instance *testingv1alpha1.TestWorker, mutateState testworker.MutateStateFn) error {
	testSuite, err := r.getTestSuite(ctx, instance)
	if err != nil {
		return err
	}

	currentState, err := r.getCurrentState(ctx, instance, testSuite)

	updatedState, err := mutateState(currentState)
	if err != nil {
		return fmt.Errorf("failed to mutate state: %w", err)
	}

	updateStateString, err := json.Marshal(updatedState)
	if err != nil {
		return fmt.Errorf("failed to marshal updated state: %w", err)
	}

	testSuite.Status.CurrentState = string(updateStateString)

	return r.Status().Update(ctx, testSuite)
}

func (r *TestWorkerReconciler) getTestSuite(ctx context.Context, instance *testingv1alpha1.TestWorker) (*testingv1alpha1.TestSuite, error) {
	testSuiteList := &testingv1alpha1.TestSuiteList{}
	if err := r.List(ctx, testSuiteList, client.InNamespace(instance.Namespace)); err != nil {
		return nil, err
	}

	if len(testSuiteList.Items) == 0 {
		return nil, fmt.Errorf("no test suite found in namespace %s", instance.Name)
	}

	return &testSuiteList.Items[0], nil
}

func (r *TestWorkerReconciler) getCurrentState(ctx context.Context, instance *testingv1alpha1.TestWorker, testSuite *testingv1alpha1.TestSuite) (interface{}, error) {
	testSuite, err := r.getTestSuite(ctx, instance)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve test suite: %w", err)
	}

	testSuiteStr := strategy.Strategy(testSuite.Spec.StateStrategy.Strategy)

	testSuiteInterface, err := testsuite.FromStrategy(&testSuiteStr, r.StrategyProviders)
	if err != nil {
		return nil, fmt.Errorf("error obtaining strategy for test suite: %w", err)
	}

	return testSuiteInterface.ParseState(testSuite.Status.CurrentState)
}
