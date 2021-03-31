package manager

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	thatchdv1alpha1 "github.com/sergioifg94/thatchd/api/v1alpha1"
	"github.com/sergioifg94/thatchd/controllers"
	"github.com/sergioifg94/thatchd/pkg/thatchd/strategy"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(thatchdv1alpha1.AddToScheme(scheme))
}

// Run starts the Thatchd manager. Applies the schemeFn to the scheme used in
// the manager client, and injects the strategyProviders in the controllers
func Run(schemeFn func(*runtime.Scheme) error, strategyProviders map[string]strategy.StrategyProvider) {
	utilruntime.Must(schemeFn(scheme))

	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "0af988fb.thatchd.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.TestSuiteReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("TestSuite"),
		Scheme:            mgr.GetScheme(),
		StrategyProviders: strategyProviders,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TestSuite")
		os.Exit(1)
	}
	if err = (&controllers.TestCaseReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("TestCase"),
		Scheme:            mgr.GetScheme(),
		StrategyProviders: strategyProviders,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TestCase")
		os.Exit(1)
	}
	if err = (&controllers.TestWorkerReconciler{
		Client:            mgr.GetClient(),
		Log:               ctrl.Log.WithName("controllers").WithName("TestWorker"),
		Scheme:            mgr.GetScheme(),
		StrategyProviders: strategyProviders,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "TestWorker")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
