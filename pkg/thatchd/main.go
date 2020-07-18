package thatchd

import (
	"errors"
	"fmt"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/sergioifg94/thatchd/pkg/apis"
	testcasecontroller "github.com/sergioifg94/thatchd/pkg/controller/testcase"
	testprogramcontroller "github.com/sergioifg94/thatchd/pkg/controller/testprogram"
	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func Run(stop <-chan struct{}, strategyProviders []strategy.StrategyProvider) error {
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		return errors.New("Failed to get watch namespace")
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return err
	}

	// Set default manager options
	options := manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	}

	mgr, err := manager.New(cfg, options)
	if err != nil {
		return err
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return err
	}

	// Add TestProgram controller
	if err := testprogramcontroller.Add(
		mgr,
		time.Second,
		strategyProviders,
	); err != nil {
		return err
	}

	if err := testcasecontroller.Add(mgr, "", strategyProviders); err != nil {
		return err
	}

	if err := mgr.Start(stop); err != nil {
		return err
	}

	return nil
}
