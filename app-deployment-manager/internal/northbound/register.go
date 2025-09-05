// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"sync"

	"buf.build/go/protovalidate"
	clientv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/fleet"
	"k8s.io/client-go/kubernetes"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openpolicyagent"
	"google.golang.org/grpc"
)

// NewDeployment creates and initializes a new deployment service.
func NewDeployment(crClient clientv1beta1.AppDeploymentClientInterface,
	opaClient openpolicyagent.ClientWithResponsesInterface,
	k8sClient *kubernetes.Clientset, fleetBundleClient *fleet.BundleClient, catalogClient catalogclient.CatalogClient, vaultAuthClient auth.VaultAuth, protoValidator ...protovalidate.Validator) *DeploymentSvc {

	var validator protovalidate.Validator
	if len(protoValidator) > 0 {
		validator = protoValidator[0]
	} else {
		// Create a default validator for backward compatibility
		validator, _ = protovalidate.New()
	}

	return &DeploymentSvc{
		crClient:          crClient,
		opaClient:         opaClient,
		fleetBundleClient: fleetBundleClient,
		k8sClient:         k8sClient,
		catalogClient:     catalogClient,
		vaultAuthClient:   vaultAuthClient,
		protoValidator:    validator,
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
	vaultAuthClient   auth.VaultAuth
	protoValidator    protovalidate.Validator
	apiMutex          sync.Mutex
}

// Register registers the gRPC services with the server
func (s *DeploymentSvc) Register(grpcServer *grpc.Server) {
	deploymentpb.RegisterDeploymentServiceServer(grpcServer, s)
	deploymentpb.RegisterClusterServiceServer(grpcServer, s)
}
