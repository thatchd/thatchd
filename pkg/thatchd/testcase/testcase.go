package testcase

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Interface interface {
	ShouldRun(testContext interface{}) bool

	Run(client client.Client) error
}
