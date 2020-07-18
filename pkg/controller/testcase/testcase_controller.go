package testcase

import (
	"context"
	"fmt"
	"time"

	thatchdv1alpha1 "github.com/sergioifg94/thatchd/pkg/apis/thatchd/v1alpha1"
	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testcase"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_testcase")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new TestCase Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, name string, strategyProviders []strategy.StrategyProvider) error {
	return add(mgr, newReconciler(mgr, strategyProviders))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, strategyProviders []strategy.StrategyProvider) reconcile.Reconciler {
	return NewReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		strategyProviders,
	)
}

func NewReconciler(client client.Client, scheme *runtime.Scheme, strategyProviders []strategy.StrategyProvider) reconcile.Reconciler {
	return &ReconcileTestCase{
		client:            client,
		scheme:            scheme,
		strategyProviders: strategyProviders,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("testcase-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TestCase
	err = c.Watch(&source.Kind{Type: &thatchdv1alpha1.TestCase{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner TestCase
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &thatchdv1alpha1.TestCase{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileTestCase implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTestCase{}

// ReconcileTestCase reconciles a TestCase object
type ReconcileTestCase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	strategyProviders []strategy.StrategyProvider
}

// Reconcile reads that state of the cluster for a TestCase object and makes changes based on the state read
// and what is in the TestCase.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTestCase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TestCase")

	// Fetch the TestCase instance
	instance := &thatchdv1alpha1.TestCase{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Test hasn't been dispatched yet
	if instance.Status.DispatchedAt == nil {
		return reconcile.Result{}, err
	}

	// Test has already started
	if instance.Status.StartedAt != nil {
		return reconcile.Result{}, err
	}

	// Update the status to mark it as running
	instance.Status.StartedAt = thatchdv1alpha1.TimeString(time.Now())
	instance.Status.Status = thatchdv1alpha1.TestCaseRunning

	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		return reconcile.Result{}, err
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

	str := strategy.Strategy(instance.Spec.Strategy)

	// Create an instance of the strategy to run the test case
	testCaseInterface, err := testcase.FromStrategy(&str, r.strategyProviders)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error obtaining strategy for test case %s: %v", instance.Name, err)
	}

	// Run the test in a goroutine and create a channel that closes when it's done
	done := make(chan error)
	go func() {
		err := testCaseInterface.Run(r.client)
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
	err = r.client.Status().Update(context.TODO(), instance)
	return reconcile.Result{}, err
}
