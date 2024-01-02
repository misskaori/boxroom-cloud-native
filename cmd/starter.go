package main

import (
	"flag"
	boxroomv1 "github.io/misskaori/boxroom-crd/api/v1"
	"github.io/misskaori/boxroom-crd/global"
	"github.io/misskaori/boxroom-crd/internal/controller"
	k8s_agent "github.io/misskaori/boxroom-crd/kubernetes/kubernetes/k8s-agent"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sync"
)

type Starter struct {
}

func (s *Starter) Init() error {
	k8sAgent, err := (&k8s_agent.ApiServerConfig{
		Url: os.Getenv("Url"),
		//AccessType: k8s_agent.InClusterConfigType,
		AccessType:       k8s_agent.KubeConfigFileType,
		KubernetesConfig: os.Getenv("KUBECONFIG"),
	}).AgentInit()

	if err != nil {
		return err
	}

	global.KubernetesAgent = k8sAgent

	return nil
}

func (s *Starter) RunBackupWatcher() {

}

func (s *Starter) RunReconciler() {
	scheme := runtime.NewScheme()
	setupLog := ctrl.Log.WithName("setup")
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(boxroomv1.AddToScheme(scheme))

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "276f5320.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controller.StorageLocationsReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "StorageLocations")
		os.Exit(1)
	}
	if err = (&controller.BackupsReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Backups")
		os.Exit(1)
	}
	if err = (&controller.RestoresReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Restores")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	group := sync.WaitGroup{}
	group.Add(1)

	go func() {
		defer group.Done()
		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}()

	group.Wait()
}
