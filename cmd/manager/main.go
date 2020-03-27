package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"runtime"

	uzap "go.uber.org/zap"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/golang/glog"

	"github.com/redhat-developer/openshift-jenkins-operator/pkg/apis"
	jenkinscontroller "github.com/redhat-developer/openshift-jenkins-operator/pkg/controller"

	appsv1 "github.com/openshift/api/apps/v1"
	imagev1 "github.com/openshift/api/image/v1"
	routev1 "github.com/openshift/api/route/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	kappsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("cmd")

func main() {
	// Add the zap logger flag set to the CLI.
	pflag.CommandLine.AddFlagSet(zap.FlagSet())
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	debug := pflag.Bool("debug", false, "Set log level to debug")

	pflag.Parse()
	logf.SetLogger(zapLogger(*debug))
	printVersion()

	namespace := ""          // namespace is set to empty string so we can watch all namespaces
	cfg := getConfigOrExit() // Get a config to talk to the apiserver
	ctx := context.TODO()    // it is still unclear which context to use, so we use TODO
	becomeLeaderOrExit(ctx)  // Become the leader before proceeding
	mgr := initializeManagerOrExit(cfg, namespace)

	log.Info("Registering Components.")
	registerComponentOrExit(mgr, apis.AddToScheme)    // Setup Scheme for all resources
	registerComponentOrExit(mgr, appsv1.AddToScheme)  // Adding the appsv1 api
	registerComponentOrExit(mgr, imagev1.AddToScheme) // Adding the imagev1 api
	registerComponentOrExit(mgr, routev1.AddToScheme) // Adding the routev1 api
	registerComponentOrExit(mgr, apis.AddToScheme)    // Setup Scheme for all resources
	registerComponentOrExit(mgr, corev1.AddToScheme)  // Adding the corev1 api
	registerComponentOrExit(mgr, kappsv1.AddToScheme) // Adding the kappsv1 api for Deployment
	log.Info("All components registered successfully.")

	// Setup all Controllers , add here other calls to your controllers
	log.Info("Registering controllers.")
	setupControllerOrExit(mgr, jenkinscontroller.AddToManager) // Setup jenkins-controller
	log.Info("All controllers registered successfully.")

	log.Info("Intializing metrics server")
	initializeMetricsServer(cfg, ctx, namespace)
	log.Info("Metrics server initialization complete.")

	log.Info("Starting the Cmd.")
	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

// Get a config to talk to the apiserver
func getConfigOrExit() *rest.Config {
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "Cannot get config to talk to the apiserver: Exiting")
		os.Exit(1)
	}
	return cfg
}

func becomeLeaderOrExit(ctx context.Context) {
	err := leader.Become(ctx, "openshift-jenkins-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
}

// Create a new Cmd to provide shared dependencies and start components
func initializeManagerOrExit(cfg *rest.Config, namespace string) manager.Manager {
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	return mgr
}

// Setup Scheme for a resources
func registerComponentOrExit(mgr manager.Manager, f func(*k8sruntime.Scheme) error) {
	// Setup Scheme for all resources
	if err := f(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Component registered: %v", reflect.ValueOf(f)))
}

// Register a controller to a manager
func setupControllerOrExit(mgr manager.Manager, f func(manager.Manager) error) {
	// Register a controller to a manager
	if err := f(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}
	log.Info(fmt.Sprintf("Controller initialized: %+v", reflect.ValueOf(f)))
}

func initializeMetricsServer(cfg *rest.Config, ctx context.Context, namespace string) {
	if err := serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}
	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}
	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

func zapLogger(debug bool) logr.Logger {
	var zapLog *uzap.Logger
	var err error
	zapLogCfg := uzap.NewDevelopmentConfig()
	if debug {
		zapLogCfg.Level = uzap.NewAtomicLevelAt(uzap.DebugLevel)
	} else {
		zapLogCfg.Level = uzap.NewAtomicLevelAt(uzap.InfoLevel)
	}
	zapLog, err = zapLogCfg.Build(uzap.AddStacktrace(uzap.DPanicLevel), uzap.AddCallerSkip(1))
	// who watches the watchmen?
	fatalIfErr(err, glog.Fatalf)
	return zapr.NewLogger(zapLog)
}

func fatalIfErr(err error, f func(format string, v ...interface{})) {
	if err != nil {
		f("unable to construct the logger: %v", err)
	}
}
