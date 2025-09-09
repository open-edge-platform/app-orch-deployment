// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"buf.build/go/protovalidate"
	"fmt"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/tenant"
	fleet2 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/k8sclient"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strconv"

	"github.com/open-edge-platform/orch-library/go/dazl"
	"google.golang.org/grpc"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	internalgrpc "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/northbound"
	utils "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/pkg/northbound"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"k8s.io/client-go/dynamic"
)

var log = dazl.GetPackageLogger()

const (
	// OIDCServerURL - address of an OpenID Connect server
	OIDCServerURL = "OIDC_SERVER_URL"
	opaHostname   = "localhost"
	OPAPort       = "OPA_PORT"
	opaScheme     = "http"
	ProjectUUID   = "MT_UPGRADE_PROJECT_ID"
)

// Config is a manager configuration
type Config struct {
	CAPath     string
	KeyPath    string
	CertPath   string
	GRPCPort   int16
	Kubeconfig string
}

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

// Start starts the NB gRPC server
func (m *Manager) Start() error {
	serverConfig := northbound.NewInsecureServerConfig(m.Config.GRPCPort)

	if oidcURL := os.Getenv(OIDCServerURL); oidcURL != "" {
		serverConfig.SecurityCfg = &northbound.SecurityConfig{
			AuthenticationEnabled: true,
			AuthorizationEnabled:  true,
		}
		log.Infof("Authentication enabled. %s=%s", OIDCServerURL, oidcURL)
		// OIDCServerURL is also referenced in jwt.go (from onos-lib-go)
	} else {
		log.Infof("Authentication not enabled %s", os.Getenv(OIDCServerURL))
	}

	var opaClient openpolicyagent.ClientWithResponsesInterface
	if serverConfig.SecurityCfg.AuthorizationEnabled && utils.IsOPAEnabled() {
		opaPortString := os.Getenv(OPAPort)
		opaPort, err := strconv.Atoi(opaPortString)
		if err != nil {
			log.Fatalf("OPA Port is no valid %v", err)
			return err
		}

		serverAddr := fmt.Sprintf("%s://%s:%d", opaScheme, opaHostname, opaPort)
		opaClient, err = openpolicyagent.NewClientWithResponses(serverAddr)
		if err != nil {
			log.Fatalf("OPA client cannot be created %v", err)
			return err
		}
		log.Infof("Authorization enabled. OPA Client=%s://%s:%d", opaScheme, opaHostname, opaPort)
	} else {
		log.Infof("Authorization not enabled")
	}

	kubeconfig := m.Config.Kubeconfig

	// TODO get actual env var from configmap once ready
	if projectUUIDVal := os.Getenv(ProjectUUID); projectUUIDVal != "" {
		// Create dynamic k8s client
		kConfig, err := clientcmd.BuildConfigFromFlags(kubeconfig, "")
		if err != nil {
			log.Fatalf("Failed to create dynamic k8s client %f", err)
			return err
		}

		kc, err := dynamic.NewForConfig(kConfig)
		if err != nil {
			log.Fatalf("Failed to create dynamic k8s client %f", err)
			return err
		}

		// Start project migration
		if err = newMigration(kc, projectUUIDVal).run(); err != nil {
			log.Fatalf("Failed to migrate project: %v", err)
		}
	} else {
		log.Infof("Project migration not enabled. %s env var not provided", ProjectUUID)
	}

	// K8s custom resource client
	crClient, err := utils.CreateClient(kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create k8s client %v", err)
		return err
	}

	// FleetBundle client
	fleetBundleClient, err := fleet2.NewBundleClient(kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create fleet bundle client %f", err)
		return err
	}

	// K8s client
	k8sClient, err := k8sclient.NewClient(kubeconfig)
	if err != nil {
		log.Fatalf("Failed to create k8s client %f", err)
		return err
	}

	// Catalog client
	catalogClient, err := catalogclient.NewCatalogClient()
	if err != nil {
		log.Fatalf("Failed to create catalog client %f", err)
		return err
	}

	// protovalidate Validator
	protoValidator, err := protovalidate.New()
	if err != nil {
		log.Fatalf("Failed to initialize protobuf validator %f", err)
		return err
	}

	// M2M auth client
	vaultAuthClient, err := auth.NewVaultAuth(utils.GetKeycloakServiceEndpoint(), utils.GetSecretServiceEndpoint(), utils.GetServiceAccount())
	if err != nil {
		log.Fatalf("Failed to create M2M auth client %f", err)
		return err
	}

	s := northbound.NewServer(serverConfig)
	s.AddService(internalgrpc.NewDeployment(crClient, opaClient, k8sClient, fleetBundleClient, catalogClient, vaultAuthClient, &protoValidator))

	msgSizeLimitBytes, err := utils.GetMessageSizeLimit()
	if err != nil {
		log.Fatalf("Failed to get msg size limit %v", err)
		return err
	}

	doneCh := make(chan error)
	go func() {
		err := s.Serve(func(_ string) {
			close(doneCh)
		}, grpc.MaxRecvMsgSize(int(msgSizeLimitBytes)))
		if err != nil {
			doneCh <- err
		}
	}()

	// Tenant event handler
	nexusHook := tenant.NewNexusHook(crClient)
	err = nexusHook.Subscribe()
	if err != nil {
		log.Fatalf("Failed to subscribe to tenant events %v", err)
		return err
	}
	log.Infof("nexus hook for MT is started")

	return <-doneCh
}
