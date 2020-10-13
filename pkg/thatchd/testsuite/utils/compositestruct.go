package utils

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/sergioifg94/thatchd/pkg/thatchd/testsuite"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CompositeStructReconciler reconciles a struct by delegating each field
// reconciliation into separate reconcilers
type CompositeStructReconciler struct {
	stateType reflect.Type
	fields    map[string]testsuite.Reconciler
}

var _ testsuite.Reconciler = &CompositeStructReconciler{}

// NewCompositeStructReconciler creates a CompositeStructReconciler for the
// stateType. Verifies that the fieldReconciler map contains entries for
// all the fields in the stateType struct
func NewCompositeStructReconciler(stateType reflect.Type, fieldReconcilers map[string]testsuite.Reconciler) (*CompositeStructReconciler, error) {
	// Validate that the type is a struct
	if stateType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("stateType must be a struct")
	}

	// Validate that there's reconcilers for every field in the struct
	numField := stateType.NumField()
	for i := 0; i < numField; i++ {
		fieldName := stateType.Field(i).Name
		if _, ok := fieldReconcilers[fieldName]; !ok {
			return nil, fmt.Errorf("no field reconciler for field %s", fieldName)
		}
	}

	return &CompositeStructReconciler{
		stateType: stateType,
		fields:    fieldReconcilers,
	}, nil
}

// Reconcile reconciles the current state by delegating the reconciliation of
// each field into the reconcilers for each field in the stateType
func (r *CompositeStructReconciler) Reconcile(client client.Client, namespace, currentStateJSON string) (interface{}, error) {
	// Initialize the current state and unmarshal it
	currentState := reflect.New(r.stateType)
	if err := json.Unmarshal([]byte(currentStateJSON), currentState.Interface()); err != nil {
		return nil, err
	}

	// Reconcile each field
	numField := r.stateType.NumField()
	for i := 0; i < numField; i++ {
		// Get the field and the field reconciler
		field := r.stateType.Field(i)
		fieldName := field.Name
		fieldReconciler, ok := r.fields[fieldName]
		if !ok {
			return nil, fmt.Errorf("no field reconciler for field %s", fieldName)
		}

		// Get the field current value
		currentFieldValue := currentState.Elem().FieldByName(fieldName)
		if currentFieldValue == (reflect.Value{}) {
			return nil, fmt.Errorf("field %s not found in state type", fieldName)
		}

		// Marshal the value to be passed to the field reconciler
		currentFieldJSON, err := json.Marshal(currentFieldValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal field %s value", fieldName)
		}

		// Reconcile the field
		reconciledFieldValue, err := fieldReconciler.Reconcile(
			client,
			namespace,
			string(currentFieldJSON),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to reconciled fiel %s: %w", fieldName, err)
		}

		// Set the field value to the reconciled one
		currentFieldValue.Set(reflect.ValueOf(reconciledFieldValue))
	}

	return currentState.Elem().Interface(), nil
}
