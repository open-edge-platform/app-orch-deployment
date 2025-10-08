// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package clusterclient

import (
	"context"
	"fmt"
	"os"

	"github.com/labstack/gommon/log"
	admapiv1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/grpc/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	activeProjectIDMetadataKey = "activeprojectid"
)

const (
	admServiceAddressEnv = "ADM_SERVICE_ADDRESS"
)

type ClientOption func(*ClientOptions)

type ClientOptions struct {
	ADMServiceAddress string `json:"admServiceAddress,omitempty"`
}

func WithADMServiceAddress(address string) ClientOption {
	return func(o *ClientOptions) {
		o.ADMServiceAddress = address
	}
}

func NewOrchClient(opts ...ClientOption) (Client, error) {
	var options ClientOptions
	for _, opt := range opts {
		opt(&options)
	}

	admServiceAddress := options.ADMServiceAddress
	if admServiceAddress == "" {
		var err error
		admServiceAddress, err = getDefaultADMServiceAddress()
		if err != nil {
			return nil, err
		}
	}

	admClusterClient, err := newADMClusterClient(admServiceAddress)
	if err != nil {
		return nil, err
	}

	vaultAuthClient, err := auth.NewVaultAuth(os.Getenv("KEYCLOAK_SERVER"), os.Getenv("VAULT_SERVER"), os.Getenv("SERVICE_ACCOUNT"))

	if err != nil {
		return nil, err
	}

	return &orchClient{
		admClusterClient: admClusterClient,
		vaultAuthClient:  vaultAuthClient,
	}, nil
}

type orchClient struct {
	admClusterClient admapiv1.ClusterServiceClient
	vaultAuthClient  auth.VaultAuth
}

func (c *orchClient) GetClusterConfig(ctx context.Context, clusterID ClusterID, projectID ProjectID) (*rest.Config, error) {
	request := &admapiv1.GetKubeConfigRequest{
		ClusterId: string(clusterID),
	}

	ctx, err := c.newCtxWithToken(ctx, projectID)
	if err != nil {
		return nil, err
	}

	resp, err := c.admClusterClient.GetKubeConfig(ctx, request)
	if err != nil {
		return nil, err
	}
	fmt.Printf("Kubeconfig: %s\n", string(resp.KubeConfigInfo.KubeConfig))
	return clientcmd.RESTConfigFromKubeConfig(resp.KubeConfigInfo.KubeConfig)
}

func (c *orchClient) newCtxWithToken(ctx context.Context, activeProjectID ProjectID) (context.Context, error) {
	token, err := c.vaultAuthClient.GetM2MToken(ctx)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, fmt.Errorf("token is empty")
	}
	return metadata.AppendToOutgoingContext(ctx,
		"authorization", "Bearer "+token,
		activeProjectIDMetadataKey, string(activeProjectID)), nil
}

func getDefaultADMServiceAddress() (string, error) {
	addr, ok := os.LookupEnv(admServiceAddressEnv)
	if !ok {
		return "", fmt.Errorf("adm service address is not set")
	}
	return addr, nil
}

func newADMClusterClient(endpoint string, opts ...grpc.DialOption) (admapiv1.ClusterServiceClient, error) {
	opts = append(opts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))
	conn, err := grpc.NewClient(endpoint, opts...)
	if err != nil {
		log.Warn(err)
		return nil, err
	}
	return admapiv1.NewClusterServiceClient(conn), nil
}
