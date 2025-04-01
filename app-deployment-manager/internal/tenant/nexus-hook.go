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
	projectActiveWatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectactivewatcher.edge-orchestrator.intel.com/v1"
	projectwatcherv1 "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/apis/projectwatcher.edge-orchestrator.intel.com/v1"
	nexus "github.com/open-edge-platform/orch-utils/tenancy-datamodel/build/nexus-client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"time"
)

var log = dazl.GetPackageLogger()

const (
	appName = "app-deployment-manager"

	nexusTimeout = 5 * time.Minute
)

type NexusHook struct {
	crClient    clientv1beta1.AppDeploymentClientInterface
	nexusClient *nexus.Clientset
}

func NewNexusHook(crClient clientv1beta1.AppDeploymentClientInterface) *NexusHook {
	return &NexusHook{
		crClient: crClient,
	}
}

func (h *NexusHook) safeUnixTime() uint64 {
	t := time.Now().Unix()
	if t < 0 {
		return 0
	}
	return uint64(t)
}

func (h *NexusHook) Subscribe() error {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("Failed to get in-cluster config: %v", err)
		return err
	}

	h.nexusClient, err = nexus.NewForConfig(cfg)
	if err != nil {
		log.Errorf("Failed to create nexus client: %v", err)
		return err
	}

	err = h.setupConfigADMWatcherConfig()
	if err != nil {
		log.Errorf("Failed to setup config ADM watcher config: %+v", err)
		return err
	}

	h.nexusClient.SubscribeAll()

	if _, err := h.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").RegisterAddCallback(h.projectCreated); err != nil {
		log.Errorf("Unable to register project creation callback: %+v", err)
		return err
	}

	if _, err := h.nexusClient.TenancyMultiTenancy().Runtime().Orgs("*").Folders("*").Projects("*").RegisterUpdateCallback(h.projectUpdated); err != nil {
		log.Errorf("Unable to register project update callback: %+v", err)
		return err
	}

	return nil
}

func (h *NexusHook) setupConfigADMWatcherConfig() error {
	tenancy := h.nexusClient.TenancyMultiTenancy()

	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	projWatcher, err := tenancy.Config().AddProjectWatchers(ctx, &projectwatcherv1.ProjectWatcher{ObjectMeta: metav1.ObjectMeta{
		Name: appName,
	}})
	if nexus.IsAlreadyExists(err) {
		log.Warnf("Project watcher already exist: appName=%s, projWatcher=%v", appName, projWatcher)
	} else if err != nil {
		log.Errorf("Failed to create project watcher: appName=%s", appName)
		return err
	}
	log.Infof("Created project watcher: appName=%s, projWatcher=%v", appName, projWatcher)
	return nil
}

func (h *NexusHook) setProjWatcherStatus(watcherObj *nexus.ProjectactivewatcherProjectActiveWatcher, statusInd projectActiveWatcherv1.ActiveWatcherStatus, message string) error {
	watcherObj.Spec.StatusIndicator = statusInd
	watcherObj.Spec.Message = message
	watcherObj.Spec.TimeStamp = h.safeUnixTime()
	log.Debugf("ProjWatcher object to update: %+v", watcherObj)

	err := watcherObj.Update(context.Background())
	if err != nil {
		log.Errorf("Failed to update ProjectActiveWatcher object with an error: %v", err)
		return err
	}
	return nil
}

func (h *NexusHook) projectCreated(project *nexus.RuntimeprojectRuntimeProject) {
	log.Infof("Creating project (no action): %v", project)
	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	// if project was deleted but recreate, we need to call update function
	if project.Spec.Deleted {
		log.Infof("Project %s is deleted, calling update function", project.DisplayName())
		h.projectUpdated(nil, project)
		return
	}

	// Register this app as an active watcher for this project.
	// As there is no action needed for the project create event, just set the status to idle and done.
	watcherObj, err := project.AddActiveWatchers(ctx, &projectActiveWatcherv1.ProjectActiveWatcher{
		ObjectMeta: metav1.ObjectMeta{
			Name: appName,
		},
		Spec: projectActiveWatcherv1.ProjectActiveWatcherSpec{
			StatusIndicator: projectActiveWatcherv1.StatusIndicationIdle,
			Message:         fmt.Sprintf("%s subscribed to project %s", appName, project.DisplayName()),
			TimeStamp:       h.safeUnixTime(),
		},
	})

	if nexus.IsAlreadyExists(err) {
		log.Warnf("Watch %s already exists for project %s", watcherObj.DisplayName(), project.DisplayName())
		return
	} else if err != nil {
		log.Errorf("Error %+v while creating watch %s for project %s", err, appName, project.DisplayName())
		return
	}

	log.Infof("Active watcher %s created for Project %s", watcherObj.DisplayName(), project.DisplayName())
}

func (h *NexusHook) projectUpdated(_, project *nexus.RuntimeprojectRuntimeProject) {
	log.Infof("Project updated: %v", project)
	if !project.Spec.Deleted {
		log.Infof("skip project update event: project is not deleted")
		return
	}

	watcherObj, err := project.GetActiveWatchers(context.Background(), appName)
	if err != nil {
		log.Errorf("Failed to get active watcher for project %s: %v", project.DisplayName(), err)
		return
	}

	err = h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationInProgress, "Deleting")
	if err != nil {
		log.Errorf("Failed to update status of ProjectActiveWatcher object: %v", err)
		// even though it is failed, keep trying to delete the project watcher
		// return
	}

	projectUID := string(project.UID)
	namespace := projectUID

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{
			string(v1beta1.AppOrchActiveProjectID): projectUID,
		},
	}

	listOptions := metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&labelSelector),
	}

	deployments, err := h.crClient.Deployments(namespace).List(context.Background(), listOptions)
	if err != nil {
		log.Warnf("cannot list deployments: %v", err)
		err := h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationError, "Failed to get deployments")
		if err != nil {
			log.Errorf("Failed to update status of ProjectActiveWatcher object: %v", err)
		}
	}
	for _, d := range deployments.Items {
		err = h.crClient.Deployments(namespace).Delete(context.Background(), d.Name, metav1.DeleteOptions{})
		if err != nil {
			log.Warnf("cannot delete deployment, it could be stale resources: %v", err)
			err := h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationError, "Failed to delete deployments")
			if err != nil {
				log.Errorf("Failed to update status of ProjectActiveWatcher object: %v", err)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), nexusTimeout)
	defer cancel()

	err = project.DeleteActiveWatchers(ctx, appName)
	if nexus.IsChildNotFound(err) {
		log.Warnf("App %s does not watch project %s", appName, project.DisplayName())
		return
	} else if err != nil {
		log.Errorf("App %s failed to delete active watchers for project %s: %v", appName, project.DisplayName(), err)
		err := h.setProjWatcherStatus(watcherObj, projectActiveWatcherv1.StatusIndicationError, "Failed to delete project")
		if err != nil {
			log.Errorf("Failed to update status of ProjectActiveWatcher object: %v", err)
		}
		return
	}
	log.Infof("App %s deleted active watchers for project %s", appName, project.DisplayName())
}
