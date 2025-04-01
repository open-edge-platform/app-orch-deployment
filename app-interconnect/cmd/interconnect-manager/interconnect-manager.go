// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	admv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/interconnect"
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/controller/network"
	skupperclient "github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/skupper/client"
	interconnectv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/interconnect/v1alpha1"
	networkv1alpha1 "github.com/open-edge-platform/app-orch-deployment/app-interconnect/pkg/apis/network/v1alpha1"
	"github.com/open-edge-platform/orch-library/go/dazl"
	_ "github.com/open-edge-platform/orch-library/go/dazl/zap"
	k8slog "github.com/open-edge-platform/orch-library/go/pkg/logging/k8s"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/runtime"
	runtimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"os"
)

var log = dazl.GetLogger()

func main() {
	logf.SetLogger(k8slog.NewControllerLogger("main"))

	cmd := getCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "interconnect-manager",
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			// Get a config to talk to the apiserver
			cfg, err := runtimeconfig.GetConfig()
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			// Create a new Cmd to provide shared dependencies and start components
			mgr, err := manager.New(cfg, manager.Options{HealthProbeBindAddress: ":8083"})
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			runtime.Must(admv1beta1.AddToScheme(mgr.GetScheme()))
			runtime.Must(interconnectv1alpha1.AddToScheme(mgr.GetScheme()))
			runtime.Must(networkv1alpha1.AddToScheme(mgr.GetScheme()))

			mode, err := cmd.Flags().GetString("mode")
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			var client clusterclient.Client
			switch mode {
			case "dev":
				client, err = clusterclient.NewLocalClient()
			case "prod":
				client, err = clusterclient.NewOrchClient()
			default:
				log.Error("invalid mode")
				os.Exit(1)
			}
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			runtime.Must(interconnect.AddControllers(mgr, client))
			runtime.Must(network.AddControllers(mgr, client))

			if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
				log.Error(err, "unable to set up health check")
				os.Exit(1)
			}
			if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
				log.Error(err, "unable to set up ready check")
				os.Exit(1)
			}
			// TODO remove this later, it is added to test build works or not
			_, err = skupperclient.NewClient("", "", "")
			if err != nil {
				log.Warn(err)
			}
			// Start the manager
			log.Info("Starting the Manager")
			if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
				log.Error(err, "controller exited non-zero")
				os.Exit(1)
			}

		},
	}
	cmd.Flags().StringP("mode", "m", "prod", "the mode in which to run the controller")
	return cmd
}
