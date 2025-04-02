// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/controllers/capi"
	"os"

	"github.com/open-edge-platform/orch-library/go/dazl"

	ctrllogger "github.com/open-edge-platform/orch-library/go/pkg/logging/k8s"
	corev1 "k8s.io/api/core/v1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/controllers/apiextension"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/controllers/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/controllers/deployment"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/controllers/deploymentcluster"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/config"
	deploymentwebhook "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/webhooks/deployment"
	//+kubebuilder:scaffold:imports

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/metrics"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	capiv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = dazl.GetPackageLogger()

	ingressKind       string
	apiProxyURL       string
	apiProxyNamespace string
	apiProxyService   string
	apiProxyPort      int
	apiGroupDomain    string
	tokenExpiryDays   string

	apiAgentChartRepo    string
	apiAgentChart        string
	apiAgentChartVersion string
	apiAgentNamespace    string
	gitCaCertFolder      string
	gitCaCertFile        string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(fleetv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(capiv1beta1.AddToScheme(scheme))

	utilruntime.Must(v1beta1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&ingressKind, "ingress-kind", "traefik", "The ingress provider kind name.")
	flag.StringVar(&apiProxyURL, "api-proxy-url", "https://service-proxy.orchestrator.intel.com",
		"The API proxy URL that API agent connects to.")
	flag.StringVar(&apiProxyNamespace, "api-proxy-namespace", "", "The namespace name that API proxy service is running.")
	flag.StringVar(&apiProxyService, "api-proxy-service", "api-proxy", "The API proxy service name.")
	flag.IntVar(&apiProxyPort, "api-proxy-port", 8123, "The API proxy service port.")
	flag.StringVar(&apiGroupDomain, "api-group-domain", "orchestrator-extension.apis", "The extension API group domain name.")
	flag.StringVar(&apiAgentChartRepo, "api-agent-chart-repo", "", "API agent helm chart repo URL")
	flag.StringVar(&apiAgentChart, "api-agent-chart", "api-agent", "API agent helm chart name")
	flag.StringVar(&apiAgentChartVersion, "api-agent-chart-version", "latest", "API agent helm chart version")
	flag.StringVar(&apiAgentNamespace, "api-agent-namespace", "orch-app", "API agent target namespace")
	flag.StringVar(&tokenExpiryDays, "token-expiry-days", "90", "Number of days after which API proxy expires registration token")
	flag.StringVar(&gitCaCertFolder, "git-ca-cert-folder", "/etc/ssl/certs/", "Folder containing the Git CA Cert file")
	flag.StringVar(&gitCaCertFile, "git-ca-cert-file", "ca.crt", "Git CA Cert file name within the Git CA Cert Folder")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(ctrllogger.NewControllerPackageLogger().WithCallDepth(2))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "cdf1d17a.edge-orchestrator.intel.com",
		WebhookServer: webhook.NewServer(
			webhook.Options{
				Port: 9443,
			}),
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&deployment.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}

	apiExtEnabled, err := utils.GetAPIExtEnabled()
	if err != nil {
		setupLog.Error(err, "controller", "APIExtension")
		os.Exit(1)
	}

	if apiExtEnabled == "true" {
		// register config first to ensure the configs are available to the controllers.
		if err := config.SetAPIExtensionConfig(&config.APIExtensionConfig{
			IngressKind:          ingressKind,
			APIProxyURL:          apiProxyURL,
			APIProxyNamespace:    apiProxyNamespace,
			APIProxyService:      apiProxyService,
			APIProxyPort:         utils.ToInt32Clamped(apiProxyPort),
			APIGroupDomain:       apiGroupDomain,
			APIAgentChartRepo:    apiAgentChartRepo,
			APIAgentChart:        apiAgentChart,
			APIAgentChartVersion: apiAgentChartVersion,
			APIAgentNamespace:    apiAgentNamespace,
			TokenExpiryDays:      tokenExpiryDays,
		}); err != nil {
			setupLog.Error(err, "unable to set config")
			os.Exit(1)
		}

		if err = (&apiextension.Reconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "APIExtension")
			os.Exit(1)
		}
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	// Start monitoring for Git Ca Cert file
	go utils.WatchGitCaCertFile(ctx, gitCaCertFolder, gitCaCertFile)

	if err = (&deploymentwebhook.Deployment{
		Client: mgr.GetClient(),
	}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Deployment")
		os.Exit(1)
	}

	if err = (&deploymentcluster.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DeploymentCluster")
		os.Exit(1)
	}

	if err = (&cluster.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Cluster")
		os.Exit(1)
	}

	capiEnabled, err := utils.GetCAPIEnableD()
	if err != nil {
		setupLog.Error(err, "controller", "CAPI")
		os.Exit(1)
	}

	if capiEnabled == "true" {
		err = capi.AddClusterController(mgr)
		if err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Cluster API Controller (CAPI)")
			os.Exit(1)
		}
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

	if err := mgr.AddMetricsServerExtraHandler("/status", promhttp.HandlerFor(
		metrics.Reg,
		promhttp.HandlerOpts{},
	)); err != nil {
		setupLog.Error(err, "unable to set up extra metrics handler")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
