package testprogram

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	thatchdv1alpha1 "github.com/sergioifg94/thatchd/pkg/apis/thatchd/v1alpha1"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testcase"
	"github.com/sergioifg94/thatchd/pkg/thatchd/testprogram"
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

var log = logf.Log.WithName("controller_testprogram")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new TestProgram Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager, reconcilePeriod time.Duration, testCases map[string]testcase.Interface, programReconciler testprogram.Reconciler) error {
	return add(mgr, newReconciler(mgr, reconcilePeriod, testCases, programReconciler))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, reconcilePeriod time.Duration, testCases map[string]testcase.Interface, programReconciler testprogram.Reconciler) reconcile.Reconciler {
	return NewReconciler(
		mgr.GetClient(),
		mgr.GetScheme(),
		reconcilePeriod,
		testCases,
		programReconciler,
	)
}

func NewReconciler(client client.Client, scheme *runtime.Scheme, reconcilePeriod time.Duration, testCases map[string]testcase.Interface, programReconciler testprogram.Reconciler) reconcile.Reconciler {
	return &ReconcileTestProgram{
		client:            client,
		scheme:            scheme,
		reconcilePeriod:   reconcilePeriod,
		testCases:         testCases,
		programReconciler: programReconciler,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("testprogram-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TestProgram
	err = c.Watch(&source.Kind{Type: &thatchdv1alpha1.TestProgram{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileTestProgram implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTestProgram{}

// ReconcileTestProgram reconciles a TestProgram object
type ReconcileTestProgram struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	reconcilePeriod time.Duration

	testCases         map[string]testcase.Interface
	programReconciler testprogram.Reconciler
}

// Reconcile reads that state of the cluster for a TestProgram object and makes changes based on the state read
// and what is in the TestProgram.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTestProgram) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TestProgram")

	// Fetch the TestProgram instance
	instance := &thatchdv1alpha1.TestProgram{}
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

	var currentState string
	if instance.Status.CurrentState != "" {
		currentState = instance.Status.CurrentState
	} else if instance.Spec.InitialState != "" {
		currentState = instance.Spec.InitialState
	} else {
		currentState = "{}"
	}

	// Reconcile the program state
	updatedState, err := r.programReconciler.Reconcile(r.client, currentState)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error reconciling program state: %v", err)
	}

	marshalledState, err := json.Marshal(updatedState)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error marshalling state: %v", err)
	}

	instance.Status.CurrentState = string(marshalledState)
	if err := r.client.Status().Update(context.TODO(), instance); err != nil {
		return reconcile.Result{}, fmt.Errorf("error updating state: %v", err)
	}

	if err := r.dispatchTestCases(reqLogger, request.Namespace, updatedState); err != nil {
		return reconcile.Result{}, fmt.Errorf("error dispatching test cases: %v", err)
	}

	return reconcile.Result{
		Requeue:      true,
		RequeueAfter: r.reconcilePeriod,
	}, nil
}

func (r *ReconcileTestProgram) dispatchTestCases(logger logr.Logger, namespace string, currentState interface{}) error {
	for testCaseName, testCaseInterface := range r.testCases {
		// Skip tests that aren't meant to be run yet
		if !testCaseInterface.ShouldRun(currentState) {
			continue
		}

		// Get the TestCase CR
		testCase := &thatchdv1alpha1.TestCase{}
		if err := r.client.Get(context.TODO(), client.ObjectKey{
			Namespace: namespace,
			Name:      testCaseName,
		}, testCase); err != nil {
			logger.Info(fmt.Sprintf("TestCase %s not found. Skipping...", testCaseName))
			continue
		}

		// Skip tests that have already been dispatched
		if testCase.Status.DispatchedAt != nil {
			continue
		}

		// Dispatch by setting the DispatchedAt field to the current time
		testCase.Status.DispatchedAt = thatchdv1alpha1.TimeString(time.Now())
		testCase.Status.Status = thatchdv1alpha1.TestCaseDispatched
		if err := r.client.Status().Update(context.TODO(), testCase); err != nil {
			return fmt.Errorf("error dispatching TestCase %s", testCaseName)
		}
	}

	return nil
}
