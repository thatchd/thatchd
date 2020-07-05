package testprogram

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler interface {
	Reconcile(client client.Client, currentState string) (interface{}, error)
}
