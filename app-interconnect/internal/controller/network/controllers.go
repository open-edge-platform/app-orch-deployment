// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = dazl.GetPackageLogger()

// AddControllers adds all network controllers to the given manager
func AddControllers(mgr manager.Manager, clusters clusterclient.Client) error {
	if err := setupFieldIndexers(mgr); err != nil {
		log.Error(err)
		return err
	}

	if err := AddDeploymentController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddDeploymentClusterController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddDeploymentServiceController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddNetworkClusterController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddNetworkClusterStatusController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddNetworkClusterLinkController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddNetworkServiceController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddNetworkLinkController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
