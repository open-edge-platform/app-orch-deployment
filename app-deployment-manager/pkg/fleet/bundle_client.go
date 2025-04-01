// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package fleet

import (
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/orch-library/go/dazl"
	fleetv1alpha1 "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/lasso/pkg/client"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type BundleClient struct {
	Client *client.Client
}

func NewBundleClient(kubeConfig string) (*BundleClient, error) {
	bundleClient := &BundleClient{}
	config, err := utils.CreateRestConfig(kubeConfig)
	if err != nil {
		log.Warnw("Failed to create REST config from kubeconfig", dazl.Error(err))
		return nil, err
	}

	newConfig := *config
	newConfig.ContentConfig.GroupVersion = &schema.GroupVersion{}
	newConfig.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	newConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	restClient, err := rest.RESTClientFor(&newConfig)
	if err != nil {
		log.Warnw("Failed to create rest client", dazl.Error(err))
		return nil, err
	}
	cl := client.NewClient(schema.GroupVersionResource{
		Group:    fleetv1alpha1.SchemeGroupVersion.Group,
		Version:  fleetv1alpha1.SchemeGroupVersion.Version,
		Resource: "bundles",
	}, "Bundle", true, restClient, 0)
	if err != nil {
		return nil, err
	}
	bundleClient.Client = cl
	return bundleClient, nil
}
