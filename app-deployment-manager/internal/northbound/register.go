// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"github.com/bufbuild/protovalidate-go"
	"sync"

	clientv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"k8s.io/client-go/kubernetes"

	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc"
)

// NewDeployment creates and initializes a new deployment service.
func NewDeployment(crClient clientv1beta1.AppDeploymentClientInterface,
	opaClient openpolicyagent.ClientWithResponsesInterface,
	k8sClient *kubernetes.Clientset, fleetBundleClient *fleet.BundleClient, catalogClient catalogclient.CatalogClient, protoValidator *protovalidate.Validator, vaultAuthClient auth.VaultAuth) *DeploymentSvc {
	return &DeploymentSvc{
		crClient:          crClient,
		opaClient:         opaClient,
		fleetBundleClient: fleetBundleClient,
		k8sClient:         k8sClient,
		catalogClient:     catalogClient,
		protoValidator:    protoValidator,
		vaultAuthClient:   vaultAuthClient,
	}
}

type DeploymentInstance struct {
	deployment         *deploymentv1beta1.Deployment
	deployments        *deploymentv1beta1.DeploymentList
	deploymentCluster  *deploymentv1beta1.DeploymentCluster
	deploymentClusters *deploymentv1beta1.DeploymentClusterList

	checkFilters []string
}

// DeploymentSvc provides deployment service functionality.
type DeploymentSvc struct {
	crClient          clientv1beta1.AppDeploymentClientInterface
	opaClient         openpolicyagent.ClientWithResponsesInterface
	fleetBundleClient *fleet.BundleClient
	k8sClient         *kubernetes.Clientset
	catalogClient     catalogclient.CatalogClient
	protoValidator    *protovalidate.Validator
	vaultAuthClient   auth.VaultAuth
	apiMutex          sync.Mutex
}

// Register is a stub method to satisfy the northbound.Service interface
// The actual API endpoints are handled by Connect-RPC in the restproxy package
func (s *DeploymentSvc) Register(_ *grpc.Server) {
	// No-op: Connect-RPC handles the API endpoints
}
