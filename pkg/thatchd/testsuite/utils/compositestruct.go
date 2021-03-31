package utils

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/thatchd/thatchd/pkg/thatchd/testsuite"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CompositeStructReconciler reconciles a struct or struct pointer by delegating
// each field reconciliation into separate reconcilers
type CompositeStructReconciler struct {
	stateType reflect.Type
	fields    map[string]testsuite.Reconciler
}

var _ testsuite.Reconciler = &CompositeStructReconciler{}

// NewCompositeStructReconciler creates a CompositeStructReconciler for the
// stateType. Verifies that the fieldReconciler map contains entries for
// all the fields in the stateType struct
func NewCompositeStructReconciler(stateType reflect.Type, fieldReconcilers map[string]testsuite.Reconciler) (*CompositeStructReconciler, error) {
	// Validate that the type is a struct or a pointer to a struct
	if stateType.Kind() != reflect.Struct {
		if stateType.Kind() != reflect.Ptr {
			return nil, fmt.Errorf("stateType must be a struct or pointer to struct")
		}

		if stateType.Elem().Kind() != reflect.Struct {
			return nil, fmt.Errorf("stateType must be a struct or pointer to struct ")
		}
	}

	// Get the struct type
	targetType := getTargetType(stateType)

	// Validate that there's reconcilers for every field in the struct
	numField := targetType.NumField()
	for i := 0; i < numField; i++ {
		fieldName := targetType.Field(i).Name
		if _, ok := fieldReconcilers[fieldName]; !ok {
			return nil, fmt.Errorf("no field reconciler for field %s", fieldName)
		}
	}

	return &CompositeStructReconciler{
		stateType: stateType,
		fields:    fieldReconcilers,
	}, nil
}

func (r *CompositeStructReconciler) ParseState(stringState string) (interface{}, error) {
	targetType := r.getTargetType()
	stateValue := reflect.New(targetType)
	if err := json.Unmarshal([]byte(stringState), stateValue.Interface()); err != nil {
		return nil, err
	}

	return stateValue.Elem().Interface(), nil
}

// Reconcile reconciles the current state by delegating the reconciliation of
// each field into the reconcilers for each field in the stateType
func (r *CompositeStructReconciler) Reconcile(client client.Client, namespace string, currentStateInterface interface{}) (interface{}, error) {
	targetType := r.getTargetType()
	result := reflect.New(targetType)
	currentState := reflect.ValueOf(currentStateInterface)

	// Reconcile each field
	numField := targetType.NumField()
	for i := 0; i < numField; i++ {
		// Get the field and the field reconciler
		field := targetType.Field(i)
		fieldName := field.Name
		fieldReconciler, ok := r.fields[fieldName]
		if !ok {
			return nil, fmt.Errorf("no field reconciler for field %s", fieldName)
		}

		// Get the field current value
		currentFieldValue := currentState.FieldByName(fieldName)
		resultFieldValue := result.Elem().FieldByName(fieldName)
		if currentFieldValue == (reflect.Value{}) {
			return nil, fmt.Errorf("field %s not found in state type", fieldName)
		}

		// Marshal the value to be passed to the field reconciler
		currentFieldJSON, err := json.Marshal(currentFieldValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("failed to marshal field %s value", fieldName)
		}

		currentField, err := fieldReconciler.ParseState(string(currentFieldJSON))
		if err != nil {
			return nil, fmt.Errorf("failed to parse field %s: %w", fieldName, err)
		}

		// Reconcile the field
		reconciledFieldValue, err := fieldReconciler.Reconcile(
			client,
			namespace,
			currentField,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to reconciled field %s: %w", fieldName, err)
		}

		// Set the field value to the reconciled one
		resultFieldValue.Set(reflect.ValueOf(reconciledFieldValue))
	}

	resultValue := result
	if r.stateType.Kind() == reflect.Struct {
		resultValue = result.Elem()
	}

	return resultValue.Interface(), nil
}

func (r *CompositeStructReconciler) getTargetType() reflect.Type {
	return getTargetType(r.stateType)
}

func getTargetType(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Ptr {
		return typ.Elem()
	}

	return typ
}
