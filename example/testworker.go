package example

import (
	"context"

	"github.com/thatchd/thatchd/pkg/thatchd/testworker"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PodAnnotationTestWorker struct {
	PodName    string
	Annotation string
	Value      string
}

var _ testworker.Interface = &PodAnnotationTestWorker{}

func (tw *PodAnnotationTestWorker) ShouldRun(s interface{}) bool {
	state := s.(PodSuiteState)
	status, ok := state[tw.PodName]

	return ok && status == PodReady
}

func (tw *PodAnnotationTestWorker) Run(ctx context.Context, namespace string, client client.Client) (testworker.MutateStateFn, error) {
	pod := &v1.Pod{}
	if err := client.Get(ctx, types.NamespacedName{
		Name:      tw.PodName,
		Namespace: namespace,
	}, pod); err != nil {
		return nil, err
	}

	pod.Annotations[tw.Annotation] = tw.Value

	err := client.Update(ctx, pod)

	return tw.setPodAnnotated, err
}

func (tw *PodAnnotationTestWorker) setPodAnnotated(i interface{}) (interface{}, error) {
	state := i.(PodSuiteState)

	state[tw.PodName] = PodAnnotated

	return state, nil
}

func NewTestWorker(config map[string]string) interface{} {
	return &PodAnnotationTestWorker{
		PodName:    config["podName"],
		Annotation: config["annotation"],
		Value:      config["value"],
	}
}
