// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"

	deploymentsv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
)

// AppDeploymentClientInterface has a method to return a DeploymentInterface.
// A group's client should implement this interface.
type AppDeploymentClientInterface interface {
	Deployments(namespace string) DeploymentInterface
	APIExtensions(namespace string) APIExtensionInterface
	GrafanaExtensions(namespace string) GrafanaExtensionInterface
	Clusters(namespace string) ClusterInterface
	DeploymentClusters(namespace string) DeploymentClusterInterface
}

// AppDeploymentClient is used to interact with features provided by the app.edge-orchestrator.intel.com group.
type AppDeploymentClient struct {
	appDeploymentClient rest.Interface
}

type Deploymentv1beta1Interface interface {
	RESTClient() rest.Interface
	AppDeploymentClientInterface
}

// Deployments returns a deploymentClient
func (a *AppDeploymentClient) Deployments(namespace string) DeploymentInterface {
	return &deploymentClient{
		apporchClient: a.appDeploymentClient,
		ns:            namespace,
	}
}

// APIExtensions returns a apiExtensionsClient
func (a *AppDeploymentClient) APIExtensions(namespace string) APIExtensionInterface {
	return &apiExtensionClient{
		appDeploymentClient: a.appDeploymentClient,
		ns:                  namespace,
	}
}

// GrafanaExtensions returns a apiExtensionsClient
func (a *AppDeploymentClient) GrafanaExtensions(namespace string) GrafanaExtensionInterface {
	return &grafanaExtensionClient{
		appDeploymentClient: a.appDeploymentClient,
		ns:                  namespace,
	}
}

// DeploymentClusters returns a deploymentClusterClient
func (a *AppDeploymentClient) DeploymentClusters(namespace string) DeploymentClusterInterface {
	return &deploymentClusterClient{
		crClient: a.appDeploymentClient,
		ns:       namespace,
	}
}

// Clusters returns a clusterClient
func (a *AppDeploymentClient) Clusters(namespace string) ClusterInterface {
	return &clusterClient{
		crClient: a.appDeploymentClient,
		ns:       namespace,
	}
}

func NewForConfig(c *rest.Config) (*AppDeploymentClient, error) {
	err := deploymentsv1beta1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	config := *c
	config.ContentConfig.GroupVersion = &deploymentsv1beta1.GroupVersion
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()

	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}

	return &AppDeploymentClient{appDeploymentClient: client}, nil
}

// NewForConfigAndClient creates a new AppDeploymentClient for the given config and http client.
// Note the http client provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *rest.Config) (*AppDeploymentClient, error) {
	config := *c

	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &AppDeploymentClient{appDeploymentClient: client}, nil
}

// New creates a new AppDeploymentClient for the given RESTClient.
func New(a rest.Interface) *AppDeploymentClient {
	return &AppDeploymentClient{a}
}

func setConfigDefaults(config *rest.Config) error {
	gv := deploymentsv1beta1.GroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (a *AppDeploymentClient) RESTClient() rest.Interface {
	if a == nil {
		return nil
	}
	return a.appDeploymentClient
}
