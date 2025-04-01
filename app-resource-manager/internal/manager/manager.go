// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/adm"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubernetes"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/kubevirt"
	resourcenbv2 "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/northbound/services/v2/resource"
	"github.com/open-edge-platform/orch-library/go/dazl"

	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/opa"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/southbound"
	envutils "github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/utils/env"
	"github.com/open-edge-platform/orch-library/go/pkg/northbound"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc"
	"os"
)

var log = dazl.GetPackageLogger()

const (
	opaHostname = "localhost"
	opaScheme   = "http"
)

// Config is a manager configuration
type Config struct {
	CAPath     string
	KeyPath    string
	CertPath   string
	GRPCPort   int16
	WSPort     int
	ConfigPath string
}

// Manager single point of entry for the provisioner
type Manager struct {
	Config Config
}

// NewManager initializes the application manager
func NewManager(cfg Config) *Manager {
	return &Manager{Config: cfg}
}

func (m *Manager) Run() {
	log.Info("Starting manager")
	if err := m.Start(); err != nil {
		log.Fatalw("Unable to start manager", dazl.Error(err))
	}
}

func (m *Manager) Start() error {
	// Create OPA client
	opaClient := opa.NewOPAClient(opaScheme, opaHostname)

	// Creates a SB handler to interact with Kubvirt API server
	admClient, err := adm.NewClient(m.Config.ConfigPath)
	if err != nil {
		log.Warn(err)
		return err
	}

	kubernetesManager := kubernetes.NewManager(m.Config.ConfigPath, admClient)
	kubevirtManager := kubevirt.NewManager(m.Config.ConfigPath, admClient, true)
	sbHandler := southbound.NewHandler(m.Config.ConfigPath, kubernetesManager, kubevirtManager)
	err = m.startNorthboundServer(sbHandler, opaClient)
	if err != nil {
		log.Warn(err)
		return err
	}
	return nil
}

// startNorthboundServer starts the northbound gRPC server
func (m *Manager) startNorthboundServer(sbHandler southbound.Handler, opaClient openpolicyagent.ClientWithResponsesInterface) error {
	serverConfig := northbound.NewInsecureServerConfig(m.Config.GRPCPort)
	if oidcURL := os.Getenv(opa.OIDCServerURL); oidcURL != "" {
		serverConfig.SecurityCfg = &northbound.SecurityConfig{
			AuthenticationEnabled: true,
			AuthorizationEnabled:  true,
		}
		log.Infow("Authentication and Authorization are enabled", dazl.String(opa.OIDCServerURL, oidcURL))
		// OIDCServerURL is also referenced in jwt.go (from lib-go)
	} else {
		log.Infow("Authentication and Authorization are not enabled", dazl.String(opa.OIDCServerURL, oidcURL))
	}

	if !serverConfig.SecurityCfg.AuthorizationEnabled {
		opaClient = nil
	}

	s := northbound.NewServer(serverConfig)
	s.AddService(resourcenbv2.NewService(sbHandler, opaClient))

	msgSizeLimitBytes, err := envutils.GetMessageSizeLimit()
	if err != nil {
		log.Warn(err)
		return err
	}

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(started string) {
			log.Infow("Started NBI on", dazl.String("address", started))
			close(doneCh)
		}, grpc.MaxRecvMsgSize(int(msgSizeLimitBytes)))
		if err != nil {
			doneCh <- err
		}
	}()
	return <-doneCh
}
