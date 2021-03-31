package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/thatchd/thatchd/pkg/thatchd/testsuite"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCompositeStructReconciler(t *testing.T) {
	compositeReconciler, err := NewCompositeStructReconciler(
		reflect.TypeOf(testStruct{}),
		map[string]testsuite.Reconciler{
			"Foo": &FooReconciler{},
			"Bar": &BarReconciler{},
		},
	)
	if err != nil {
		t.Fatalf("unexpected error instantiating reconciler: %v", err)
	}

	client := fake.NewFakeClient()
	currentStateJSON := `
	{
		"foo": {
			"A": "hello",
			"B": "world"
		},
		"bar": {
			"A": 42,
			"B": 24
		}
	}
	`

	currentState, err := compositeReconciler.ParseState(currentStateJSON)
	if err != nil {
		t.Fatal(err)
	}

	result, err := compositeReconciler.Reconcile(client, "", currentState)
	if err != nil {
		t.Fatal(err)
	}

	resultTyped, ok := result.(testStruct)
	if !ok {
		t.Fatal("result type not expected")
	}

	expectedResult := testStruct{
		Foo: &foo{
			A: "hello foo",
			B: "foo world",
		},
		Bar: &bar{
			A: 43,
			B: 25,
		},
	}

	if !reflect.DeepEqual(resultTyped, expectedResult) {
		t.Errorf("unmatching resulting value. Expected %v, got %v", expectedResult, resultTyped)
	}
}

type foo struct {
	A string
	B string
}

type bar struct {
	A int
	B int
}

type testStruct struct {
	Foo *foo `json:"foo"`
	Bar *bar `json:"bar"`
}

type FooReconciler struct{}
type BarReconciler struct{}

var _ testsuite.Reconciler = &FooReconciler{}
var _ testsuite.Reconciler = &BarReconciler{}

func (r *FooReconciler) ParseState(state string) (interface{}, error) {
	currentState := &foo{}
	if err := json.Unmarshal([]byte(state), currentState); err != nil {
		return nil, fmt.Errorf("Failed to unmarshall state %s: %w", state, err)
	}

	return currentState, nil
}
func (r *FooReconciler) Reconcile(_ client.Client, _ string, currentState interface{}) (interface{}, error) {
	return &foo{
		A: fmt.Sprintf("%s foo", currentState.(*foo).A),
		B: fmt.Sprintf("foo %s", currentState.(*foo).B),
	}, nil
}
func (r *BarReconciler) ParseState(state string) (interface{}, error) {
	currentState := &bar{}
	if err := json.Unmarshal([]byte(state), currentState); err != nil {
		return nil, fmt.Errorf("Failed to unmarshall state %s: %w", state, err)
	}

	return currentState, nil
}
func (r *BarReconciler) Reconcile(_ client.Client, _ string, currentState interface{}) (interface{}, error) {
	return &bar{
		A: currentState.(*bar).A + 1,
		B: currentState.(*bar).B + 1,
	}, nil
}
