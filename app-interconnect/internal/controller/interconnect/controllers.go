// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package interconnect

import (
	"github.com/open-edge-platform/app-orch-deployment/app-interconnect/internal/cluster"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = dazl.GetPackageLogger()

// AddControllers adds all network controllers to the given manager
func AddControllers(mgr manager.Manager, clusters clusterclient.Client) error {
	if err := AddClusterController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddClusterStatusController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddLinkController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	if err := AddServiceController(mgr, clusters); err != nil {
		log.Error(err)
		return err
	}
	return nil
}
