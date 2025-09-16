// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"fmt"
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
// Returns an error if protobuf validator creation fails.
func NewDeployment(crClient clientv1beta1.AppDeploymentClientInterface,
	opaClient openpolicyagent.ClientWithResponsesInterface,
	k8sClient *kubernetes.Clientset, fleetBundleClient *fleet.BundleClient, catalogClient catalogclient.CatalogClient, vaultAuthClient auth.VaultAuth, protoValidator *protovalidate.Validator) (*DeploymentSvc, error) {

	var validator protovalidate.Validator
	var err error

	if protoValidator != nil {
		// validate the validator it catch cases where an uninitialized validator is passed
		defer func() {
			if r := recover(); r != nil {
				// If we get a panic from the validator, it's likely uninitialized
				err = fmt.Errorf("provided protobuf validator appears to be uninitialized: %v", r)
			}
		}()
		validator = *protoValidator
	} else {
		// Create a default validator for backward compatibility
		validator, err = protovalidate.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create protobuf validator: %w", err)
		}
	}

	// Check if there was a panic during validator assignment
	if err != nil {
		return nil, err
	}

	return &DeploymentSvc{
		crClient:          crClient,
		opaClient:         opaClient,
		fleetBundleClient: fleetBundleClient,
		k8sClient:         k8sClient,
		catalogClient:     catalogClient,
		vaultAuthClient:   vaultAuthClient,
		protoValidator:    validator,
	}, nil
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

// NewDeploymentMustSucceed is a convenience wrapper for tests that panics on validator creation failure.
// This maintains backward compatibility with existing test code.
// For production code, use NewDeployment which returns an error.
func NewDeploymentMustSucceed(crClient clientv1beta1.AppDeploymentClientInterface,
	opaClient openpolicyagent.ClientWithResponsesInterface,
	k8sClient *kubernetes.Clientset, fleetBundleClient *fleet.BundleClient, catalogClient catalogclient.CatalogClient, vaultAuthClient auth.VaultAuth, protoValidator *protovalidate.Validator) *DeploymentSvc {

	svc, err := NewDeployment(crClient, opaClient, k8sClient, fleetBundleClient, catalogClient, vaultAuthClient, protoValidator)
	if err != nil {
		panic(fmt.Sprintf("NewDeployment failed in test: %v", err))
	}
	return svc
}

// Register registers the gRPC services with the server
func (s *DeploymentSvc) Register(grpcServer *grpc.Server) {
	deploymentpb.RegisterDeploymentServiceServer(grpcServer, s)
	deploymentpb.RegisterClusterServiceServer(grpcServer, s)
}
