// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package admclient

import (
	"context"
	"fmt"
	"os"

	admv1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	clusterv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	clientv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/grpc/retry"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type ADMClient interface {
	GetClusterToken(ctx context.Context, clusterID, namespace, serviceAccount string, expiration *int64) (string, error)
	GetClusterInfraName(ctx context.Context, clusterID, activeProjectID string) (string, error)
}

type client struct {
	clusterServiceClient admv1.ClusterServiceClient
	vaultAuthClient      auth.VaultAuth
	appDepClient         *clientv1beta1.AppDeploymentClient
}

func (c *client) GetClusterToken(ctx context.Context, clusterID, namespace, serviceAccount string, expiration *int64) (string, error) {
	ctx, cancel, err := getCtxWithToken(ctx, c.vaultAuthClient)
	if err != nil {
		return "", err
	}
	defer cancel()

	req := &admv1.GetKubeConfigRequest{ClusterId: clusterID}

	resp, err := c.clusterServiceClient.GetKubeConfig(ctx, req)
	if err != nil {
		return "", err
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(resp.KubeConfigInfo.KubeConfig)
	if err != nil {
		return "", err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", err
	}

	tokenReq := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			Audiences:         []string{},
			ExpirationSeconds: expiration,
		},
	}
	tokenCreated, err := clientset.CoreV1().ServiceAccounts(namespace).
		CreateToken(ctx, serviceAccount, tokenReq, v1.CreateOptions{})
	if err != nil || tokenCreated.Status.Token == "" {
		return "", fmt.Errorf("failed to create token %v", err)
	}

	return tokenCreated.Status.Token, nil
}

func (c *client) GetClusterInfraName(ctx context.Context, clusterID, activeProjectID string) (string, error) {
	namespace := activeProjectID
	clusterClient := c.appDepClient.Clusters(namespace)
	labelSelector := v1.LabelSelector{
		MatchLabels: map[string]string{string(clusterv1beta1.ClusterName): clusterID},
	}

	if activeProjectID != "" {
		labelSelector.MatchLabels[string(clusterv1beta1.AppOrchActiveProjectID)] = activeProjectID
	}

	listOpts := v1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	clusters, err := clusterClient.List(ctx, listOpts)
	if err != nil {
		logrus.Warn("cluster client list err : ", err)
		return "", err
	}

	var capiInfraName string
	if clusters != nil && len(clusters.Items) > 0 {
		for i := 0; i < len(clusters.Items); i++ {
			// Access labels
			labels := clusters.Items[i].GetLabels()
			if labels != nil {
				_, exists := labels[string(clusterv1beta1.CapiInfraName)]
				if exists {
					capiInfraName = clusters.Items[i].ObjectMeta.Labels[string(clusterv1beta1.CapiInfraName)]
					logrus.Debug("capiInfraName : ", capiInfraName)
					break
				}
			} else {
				logrus.Warn("No labels found")
			}
		}
	}
	return capiInfraName, nil
}

var NewClient = func(opts ...grpc.DialOption) (ADMClient, error) {
	admAddr := os.Getenv("ADM_ADDRESS")
	if admAddr == "" {
		return nil, fmt.Errorf("ADM_ADDRESS is not set")
	}

	opts = append(opts,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(retry.RetryingStreamClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))),
		grpc.WithUnaryInterceptor(retry.RetryingUnaryClientInterceptor(retry.WithRetryOn(codes.Unavailable, codes.Unknown))))
	// nolint:staticcheck
	conn, err := grpc.DialContext(context.Background(), admAddr, opts...)
	if err != nil {
		return nil, err
	}

	// M2M auth client
	vaultAuthClient, err := auth.NewVaultAuth(utils.GetKeycloakServiceEndpoint(), utils.GetSecretServiceEndpoint(), utils.GetServiceAccount())
	if err != nil {
		return nil, err
	}

	clusterServiceClient := admv1.NewClusterServiceClient(conn)
	appDepClient, err := utils.CreateClient("")
	if err != nil {
		logrus.Warn("create App deployment client failed : ", err)
		return nil, err
	}

	cl := &client{
		clusterServiceClient: clusterServiceClient,
		vaultAuthClient:      vaultAuthClient,
		appDepClient:         appDepClient,
	}

	return cl, nil
}
