// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package tenant

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	clientv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/open-edge-platform/orch-library/go/dazl"
	"github.com/open-edge-platform/orch-library/go/pkg/tenancy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var log = dazl.GetPackageLogger()

// Handler implements tenancy.Handler for the app-deployment-manager.
// It handles project lifecycle events: create is a no-op, delete cleans
// up AppDeployment CRs in the project namespace.
type Handler struct {
	crClient clientv1beta1.AppDeploymentClientInterface
}

// NewHandler creates a new tenancy event handler.
func NewHandler(crClient clientv1beta1.AppDeploymentClientInterface) *Handler {
	return &Handler{
		crClient: crClient,
	}
}

// HandleEvent processes a single tenancy event. Only project events are
// handled; org events are ignored.
func (h *Handler) HandleEvent(ctx context.Context, event tenancy.Event) error {
	if event.ResourceType != "project" {
		return nil // only project events are relevant
	}

	switch event.EventType {
	case "created":
		log.Infof("Project created (no action): %s", event.ResourceName)
		return nil

	case "deleted":
		return h.handleProjectDeleted(ctx, event)

	default:
		log.Infof("Ignoring unknown event type %q for project %s", event.EventType, event.ResourceName)
		return nil
	}
}

// handleProjectDeleted removes all AppDeployment CRs in the project namespace.
func (h *Handler) handleProjectDeleted(ctx context.Context, event tenancy.Event) error {
	log.Infof("Project deleted, cleaning up deployments: %s (id=%s)", event.ResourceName, event.ResourceID)

	projectUID := event.ResourceID.String()
	namespace := projectUID

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			string(v1beta1.AppOrchActiveProjectID): projectUID,
		},
	}

	listOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&labelSelector),
	}

	deployments, err := h.crClient.Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		return fmt.Errorf("cannot list deployments in namespace %s: %w", namespace, err)
	}

	var lastErr error
	for _, d := range deployments.Items {
		if err := h.crClient.Deployments(namespace).Delete(ctx, d.Name, metav1.DeleteOptions{}); err != nil {
			log.Warnf("cannot delete deployment %s, it could be stale resources: %v", d.Name, err)
			lastErr = err
		}
	}

	if lastErr != nil {
		return fmt.Errorf("failed to delete some deployments in namespace %s: %w", namespace, lastErr)
	}

	log.Infof("Cleaned up deployments for project %s", event.ResourceName)
	return nil
}
