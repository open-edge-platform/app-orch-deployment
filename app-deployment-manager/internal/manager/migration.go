// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"context"

	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	appOrchResources = []string{"deployments", "clusters", "deploymentclusters"}
	rancherResources = []string{"gitrepos", "clusters", "bundles"}
)

type Migration struct {
	k8sClient *dynamic.DynamicClient
	newUUID   string
	fleetGVR  schema.GroupVersionResource
}

func newMigration(k8sClient *dynamic.DynamicClient, newUUID string) *Migration {
	return &Migration{k8sClient: k8sClient, newUUID: newUUID}
}

func (m *Migration) run() error {
	log.Infof("Running project migration to %s", m.newUUID)

	fleetGVR := schema.GroupVersionResource{
		Group:    "app.edge-orchestrator.intel.com",
		Version:  "v1beta1",
		Resource: "",
	}

	m.fleetGVR = fleetGVR

	if err := m.start(context.Background()); err != nil {
		return err
	}

	log.Infof("Completed project migration to %s", m.newUUID)
	return nil
}

func (m *Migration) start(ctx context.Context) error {
	// Start AppOrch resource migration
	for _, r := range appOrchResources {
		if err := m.migrate(ctx, r); err != nil {
			return err
		}
	}

	// Start Rancher resource migration
	m.fleetGVR.Group = "fleet.cattle.io"
	m.fleetGVR.Version = "v1alpha1"

	for _, r := range rancherResources {
		if err := m.migrate(ctx, r); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migration) migrate(ctx context.Context, resource string) error {
	m.fleetGVR.Resource = resource
	crList, err := m.k8sClient.Resource(m.fleetGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(crList.Items) == 0 {
		log.Infof("%s CR: no resources deployed", resource)
		return nil
	}

	// Update CR
	for _, cr := range crList.Items {
		foundLabel := false
		labelList := cr.GetLabels()

		// Skip if label already present
		for l := range labelList {
			if l == string(deploymentv1beta1.AppOrchActiveProjectID) {
				foundLabel = true
				// break since no need to loop thru remaining labels
				break
			}
		}

		if !(foundLabel) {
			labelList[string(deploymentv1beta1.AppOrchActiveProjectID)] = m.newUUID
			cr.SetLabels(labelList)

			_, err := m.k8sClient.Resource(m.fleetGVR).Namespace(cr.GetNamespace()).Update(ctx, &cr, metav1.UpdateOptions{})
			if err != nil {
				log.Warnf("%s CR: cannot update labels to migrate: %v", resource, err)
				return errors.Status(k8serrors.K8sToTypedError(err)).Err()
			}
			log.Infof(`%s CR: "%s" label added | %s: %s`, resource, cr.GetName(), string(deploymentv1beta1.AppOrchActiveProjectID), m.newUUID)
		} else {
			log.Infof(`%s CR: "%s" label already present | %s: %s`, resource, cr.GetName(), string(deploymentv1beta1.AppOrchActiveProjectID), labelList[string(deploymentv1beta1.AppOrchActiveProjectID)])
		}
	}

	return nil
}
