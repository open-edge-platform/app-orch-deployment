// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package adm

import (
	"context"
	clusterapi "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-resource-manager/internal/model"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/grpc/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

var log = dazl.GetPackageLogger()

//go:generate mockery --name Client --filename adm_client_mock.go --structname MockADMClient
type Client interface {
	GetKubeConfig(ctx context.Context, clusterID string) ([]byte, error)
	GetAppNamespace(ctx context.Context, appID string) (string, error)
}

type client struct {
	clusterServiceClient    clusterapi.ClusterServiceClient
	deploymentServiceClient clusterapi.DeploymentServiceClient
	vaultAuthClient         auth.VaultAuth
}

// GetAppNamespace gets application namespace
func (c *client) GetAppNamespace(ctx context.Context, appID string) (string, error) {
	ctx, cancel, err := addToOutgoingContext(ctx, c.vaultAuthClient, true)
	if err != nil {
		log.Warn(err)
		return "", err
	}
	defer cancel()

	resp, err := c.deploymentServiceClient.GetAppNamespace(ctx, &clusterapi.GetAppNamespaceRequest{
		AppId: appID,
	})
	if err != nil {
		log.Warn(err)
		return "", err
	}
	return resp.Namespace, nil
}

// GetKubeConfig gets kubeconfig based on a given cluster ID
func (c *client) GetKubeConfig(ctx context.Context, clusterID string) ([]byte, error) {
	ctx, cancel, err := addToOutgoingContext(ctx, c.vaultAuthClient, true)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	defer cancel()

	request := &clusterapi.GetKubeConfigRequest{
		ClusterId: clusterID,
	}

	resp, err := c.clusterServiceClient.GetKubeConfig(ctx, request)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	return resp.KubeConfigInfo.KubeConfig, nil
}

// NewClient creates a new ADM client
func NewClient(configPath string, opts ...grpc.DialOption) (Client, error) {
	config, err := model.GetConfigModel(configPath)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	opts = append(opts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))
	// nolint:staticcheck
	conn, err := grpc.DialContext(context.Background(), config.AppDeploymentManager.Endpoint, opts...)
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	clusterServiceClient := clusterapi.NewClusterServiceClient(conn)
	deploymntServiceClient := clusterapi.NewDeploymentServiceClient(conn)

	// M2M auth client
	vaultAuthClient, err := auth.NewVaultAuth(os.Getenv("KEYCLOAK_SERVER"), os.Getenv("VAULT_SERVER"), os.Getenv("SERVICE_ACCOUNT"))
	if err != nil {
		log.Warn(err)
		return nil, err
	}

	cl := &client{
		clusterServiceClient:    clusterServiceClient,
		deploymentServiceClient: deploymntServiceClient,
		vaultAuthClient:         vaultAuthClient,
	}

	return cl, nil

}

var _ Client = &client{}
