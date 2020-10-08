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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	thatchdv1alpha1 "github.com/sergioifg94/thatchd/api/v1alpha1"
	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testcase"
)

// TestCaseReconciler reconciles a TestCase object
type TestCaseReconciler struct {
	client.Client
	Log               logr.Logger
	Scheme            *runtime.Scheme
	StrategyProviders []strategy.StrategyProvider
}

// +kubebuilder:rbac:groups=testing.thatchd.io,resources=testcases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=testing.thatchd.io,resources=testcases/status,verbs=get;update;patch

func (r *TestCaseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("testcase", req.NamespacedName)

	// Fetch the TestCase instance
	instance := &thatchdv1alpha1.TestCase{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Test hasn't been dispatched yet
	if instance.Status.DispatchedAt == nil {
		return ctrl.Result{}, err
	}

	// Test has already started
	if instance.Status.StartedAt != nil {
		return ctrl.Result{}, err
	}

	// Update the status to mark it as running
	instance.Status.StartedAt = thatchdv1alpha1.TimeString(time.Now())
	instance.Status.Status = thatchdv1alpha1.TestCaseRunning

	if err := r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}

	// Get the timeout channel, or a channel that never closes if timeout hasn't
	// been specified
	var timeoutCh <-chan time.Time
	if instance.Spec.Timeout != nil {
		timeout, _ := time.ParseDuration(*instance.Spec.Timeout)
		timeoutCh = time.After(timeout)
	} else {
		timeoutCh = make(<-chan time.Time)
	}

	str := strategy.Strategy(instance.Spec.Strategy.Strategy)

	// Create an instance of the strategy to run the test case
	testCaseInterface, err := testcase.FromStrategy(&str, r.StrategyProviders)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error obtaining strategy for test case %s: %v", instance.Name, err)
	}

	// Run the test in a goroutine and create a channel that closes when it's done
	done := make(chan error)
	go func() {
		err := testCaseInterface.Run(r, req.Namespace)
		done <- err
	}()

	var testError error
	var testCaseStatus = thatchdv1alpha1.TestCaseFinished

	// Block until either the timeout or the done channel emit, and depending
	// on which channel, set the new values for the status
	select {
	case <-timeoutCh:
		testError = fmt.Errorf("test timed out after %v", instance.Spec.Timeout)
		testCaseStatus = thatchdv1alpha1.TestCaseCanceled
	case err := <-done:
		testError = err
		if err != nil {
			testCaseStatus = thatchdv1alpha1.TestCaseFailed
		}
	}

	// If an error occurred set the failure message
	if testError != nil {
		failureMessage := testError.Error()
		instance.Status.FailureMessage = &failureMessage
	}
	instance.Status.Status = testCaseStatus
	instance.Status.FinishedAt = thatchdv1alpha1.TimeString(time.Now())

	// Update the CR status
	err = r.Status().Update(context.TODO(), instance)
	return ctrl.Result{}, err
}

func (r *TestCaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&thatchdv1alpha1.TestCase{}).
		Complete(r)
}
