// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	yaml2 "sigs.k8s.io/yaml"

	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/vault"
)

const (
	MaskedValuePlaceholder = "********"
)

const maxDepth = 20

func deploymentType(s string) deploymentv1beta1.DeploymentType {
	switch s := deploymentv1beta1.DeploymentType(s); s {
	case deploymentv1beta1.AutoScaling:
		return deploymentv1beta1.AutoScaling
	case deploymentv1beta1.Targeted:
		return deploymentv1beta1.Targeted
	default:
		return deploymentv1beta1.AutoScaling
	}
}

func deploymentState(s string) deploymentpb.State {
	switch s {
	case "True", "Running":
		return deploymentpb.State_RUNNING
	case "False", "Down":
		return deploymentpb.State_DOWN
	case "InternalError":
		return deploymentpb.State_INTERNAL_ERROR
	case "Deploying":
		return deploymentpb.State_DEPLOYING
	case "Updating":
		return deploymentpb.State_UPDATING
	case "Terminating":
		return deploymentpb.State_TERMINATING
	case "Error":
		return deploymentpb.State_ERROR
	case "NoTargetClusters":
		return deploymentpb.State_NO_TARGET_CLUSTERS
	case "Unknown":
		return deploymentpb.State_UNKNOWN
	default:
		return deploymentpb.State_DEPLOYING
	}
}

// labelsMatch checks if two label maps are exactly identical.
// Returns true only if both maps have the same keys and values.
func labelsMatch(labels1, labels2 map[string]string) bool {
	if len(labels1) != len(labels2) {
		return false
	}
	for key, value := range labels1 {
		if labels2Value, exists := labels2[key]; !exists || labels2Value != value {
			return false
		}
	}
	return true
}

// Set the details of the deployment and return the instance.
func createDeploymentCr(d *Deployment, scenario string, resourceVersion string, existingDeployment *deploymentv1beta1.Deployment) (*deploymentv1beta1.Deployment, error) {
	labelList := map[string]string{
		"app.kubernetes.io/name":       "deployment",
		"app.kubernetes.io/instance":   d.Name,
		"app.kubernetes.io/part-of":    "app-deployment-manager",
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "app-deployment-manager",
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)

	labelList[activeProjectIDKey] = d.ActiveProjectID

	// For update scenario, preserve NetworkRef if not provided in the request
	if scenario == "update" && existingDeployment != nil {
		if d.NetworkName == "" && existingDeployment.Spec.NetworkRef.Name != "" {
			d.NetworkName = existingDeployment.Spec.NetworkRef.Name
		}
	}

	// fixme: understand where namespaceLabel is retrieved from ie controller, fleet ?
	namespaceLabels := map[string]string{}

	var targetNamespace string

	rsRepoURL := os.Getenv("RS_PROXY_REPO")
	rsRepoSec := os.Getenv("RS_PROXY_REPO_SECRET")

	applicationsList := make([]deploymentv1beta1.Application, len(*d.HelmApps))
	if len(*d.HelmApps) != 0 {
		for appsIndex, app := range *d.HelmApps {
			targetsList := make([]map[string]string, 0)
			enabledServiceExport := false
			for _, serviceExport := range d.ServiceExports {
				if app.Name == serviceExport.AppName {
					enabledServiceExport = true
					break
				}
			}

			// For both AutoScaling and Targeted: UI sends complete desired state during UPDATE
			// No need to accumulate - just process the targets from the request

			// Add or update targets from the request
			for _, target := range d.TargetClusters {
				if app.Name == target.AppName {
					if d.DeploymentType == string(deploymentv1beta1.Targeted) {
						target.Labels[string(deploymentv1beta1.ClusterName)] = target.ClusterId
					}

					// For AutoScaling: Support multiple expansion strategies for OR logic:
					// 1. Comma-separated values: {"host": "ubuntu,emt"} → [{"host":"ubuntu"}, {"host":"emt"}]
					// 2. Multiple different keys: {"host": "ubuntu", "test": "app"} → [{"host":"ubuntu"}, {"test":"app"}]
					// This allows matching clusters that have ANY of the labels, not ALL of them
					if d.DeploymentType == string(deploymentv1beta1.AutoScaling) {
						projectIDKey := string(deploymentv1beta1.FleetProjectID)

						// First, check for comma-separated values and expand them
						expandedTargets := []map[string]string{}
						hasCommaSeparated := false

						for key, value := range target.Labels {
							if key != projectIDKey && strings.Contains(value, ",") {
								hasCommaSeparated = true
								// Split comma-separated values into separate targets
								values := strings.Split(value, ",")
								for _, v := range values {
									v = strings.TrimSpace(v)
									if v != "" {
										labelMap := make(map[string]string)
										labelMap[projectIDKey] = target.Labels[projectIDKey]
										labelMap[key] = v
										expandedTargets = append(expandedTargets, labelMap)
									}
								}
								break // Only support one comma-separated key at a time
							}
						}

						if hasCommaSeparated {
							// Add all expanded targets
							for _, labelSet := range expandedTargets {
								isDuplicate := false
								for _, existing := range targetsList {
									if labelsMatch(existing, labelSet) {
										isDuplicate = true
										break
									}
								}
								if !isDuplicate {
									targetsList = append(targetsList, labelSet)
								}
							}
							continue // Skip the normal processing below
						}
						hasMultipleLabels := false

						// Check if target has multiple non-project-id labels
						for key := range target.Labels {
							if key != projectIDKey {
								if hasMultipleLabels {
									// Found second non-project label, needs splitting
									break
								}
								hasMultipleLabels = true
							}
						}

						// Only split if there are multiple non-project-id labels
						labelCount := 0
						for key := range target.Labels {
							if key != projectIDKey {
								labelCount++
							}
						}

						if labelCount > 1 {
							var nonProjectLabels []map[string]string

							for key, value := range target.Labels {
								if key != projectIDKey {
									labelMap := make(map[string]string)
									labelMap[projectIDKey] = target.Labels[projectIDKey] // Always include project ID
									labelMap[key] = value
									nonProjectLabels = append(nonProjectLabels, labelMap)
								}
							}

							// Add each label as a separate target entry (OR logic)
							if len(nonProjectLabels) > 0 {
								for _, labelSet := range nonProjectLabels {
									// Check if this exact label set already exists
									isDuplicate := false
									for _, existing := range targetsList {
										if labelsMatch(existing, labelSet) {
											isDuplicate = true
											break
										}
									}
									if !isDuplicate {
										targetsList = append(targetsList, labelSet)
									}
								}
								continue // Skip the normal processing below
							}
						}
					}

					// Check if this target overlaps with any existing target
					targetExists := false
					overlappingTargetIndex := -1

					// For Targeted deployments, match by ClusterName and merge metadata
					if d.DeploymentType == string(deploymentv1beta1.Targeted) {
						targetClusterName := target.Labels[string(deploymentv1beta1.ClusterName)]
						for i, existingTarget := range targetsList {
							existingClusterName := existingTarget[string(deploymentv1beta1.ClusterName)]
							if existingClusterName == targetClusterName && targetClusterName != "" {
								targetExists = true
								overlappingTargetIndex = i
								break
							}
						}

						if targetExists {
							// Merge new metadata into existing target
							for key, value := range target.Labels {
								targetsList[overlappingTargetIndex][key] = value
							}
						}
					} else {
						// For AutoScaling deployments:
						// Each target is a separate label selector that should be preserved independently
						// Simply check if this exact target already exists to avoid duplicates

						// Check if this exact target already exists
						exactMatchFound := false
						for i := range targetsList {
							if labelsMatch(targetsList[i], target.Labels) {
								exactMatchFound = true
								targetExists = true
								break
							}
						}

						// For UPDATE: If not an exact match, it's a new target - add it
						// For CREATE: Will be added below if !targetExists
						if !exactMatchFound {
							targetExists = false
						}
					}

					// Add new target if it's a completely new target (not an update scenario)
					if !targetExists {
						targetsList = append(targetsList, target.Labels)
					}
				}
			}

			targetNamespace = app.DefaultNamespace
			for _, override := range d.OverrideValues {
				if app.Name == override.AppName {
					if override.TargetNamespace != "" {
						targetNamespace = override.TargetNamespace
					}
					break
				}
			}

			if targetNamespace == "" {
				// set namespace to unique deployment name if no default
				// namespace is set by user
				targetNamespace = d.Name
			}

			var ignoreResource []deploymentv1beta1.IgnoreResource

			for _, ign := range app.IgnoreResource {
				ignoreResource = append(ignoreResource, deploymentv1beta1.IgnoreResource{
					Name:      ign.Name,
					Kind:      ign.Kind,
					Namespace: ign.Namespace,
				})
			}

			var repoSecretName string
			if app.Repo != "" && app.Repo == rsRepoURL {
				repoSecretName = rsRepoSec
			} else {
				repoSecretName = d.RepoSecretName[app.Name]
			}

			// for dependency support
			// add DependentDeploymentpackageList here
			dependentDeploymentPackages := make(map[string]deploymentv1beta1.DeploymentPackageRef)

			// since each item in app.RequiredDeploymentPackages does not have "ForbidsMultipleDeployments" information,
			// need to retrieve it from this Deployment object
			requiredDPMap := make(map[string]*deploymentv1beta1.DeploymentPackageRef)
			for _, deplReqDP := range d.RequiredDeploymentPackage {
				requiredDPMap[catalogclient.GetDeploymentPackageID(deplReqDP.Name, deplReqDP.Version, deplReqDP.ProfileName)] = deplReqDP
			}
			for _, appReqDP := range app.RequiredDeploymentPackages {
				dependentDeploymentPackages[appReqDP.GetID()] = deploymentv1beta1.DeploymentPackageRef{
					Name:                       appReqDP.Name,
					Version:                    appReqDP.Version,
					ProfileName:                appReqDP.Profile,
					ForbidsMultipleDeployments: requiredDPMap[appReqDP.GetID()].ForbidsMultipleDeployments,
				}
			}

			applicationsList[appsIndex] = deploymentv1beta1.Application{
				Name:                app.Name,
				Version:             d.AppVersion,
				Namespace:           targetNamespace,
				NamespaceLabels:     namespaceLabels,
				Targets:             targetsList,
				EnableServiceExport: enabledServiceExport,
				ProfileSecretName:   d.ProfileSecretName[app.Name],
				ValueSecretName:     d.ValueSecretName[app.Name],
				DependsOn:           app.DependsOn,
				RedeployAfterUpdate: app.RedeployAfterUpdate,
				IgnoreResources:     ignoreResource,
				HelmApp: &deploymentv1beta1.HelmApp{
					Chart:                   app.Chart,
					Version:                 app.Version,
					Repo:                    app.Repo,
					RepoSecretName:          repoSecretName,
					ImageRegistry:           app.ImageRegistry,
					ImageRegistrySecretName: d.ImageRegistrySecretName[app.Name],
				},
				DependentDeploymentPackages: dependentDeploymentPackages,
			}
		}
	}

	dpRef := deploymentv1beta1.DeploymentPackageRef{
		Name:                       d.AppName,
		Version:                    d.AppVersion,
		ProfileName:                d.ProfileName,
		ForbidsMultipleDeployments: d.ForbidsMultipleDeployments,
		Namespaces:                 d.Namespaces,
	}

	if scenario == "create" {
		childDeploymentList := make(map[string]deploymentv1beta1.DependentDeploymentRef)

		setInstance := &deploymentv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      d.Name,
				Namespace: d.Namespace,
				Labels:    labelList,
			},
			Spec: deploymentv1beta1.DeploymentSpec{
				DisplayName:          d.DisplayName,
				Project:              d.Project,
				DeploymentPackageRef: dpRef,
				Applications:         applicationsList,
				DeploymentType:       deploymentType(d.DeploymentType),
				NetworkRef: corev1.ObjectReference{
					Name:       d.NetworkName,
					Kind:       "Network",
					APIVersion: "network.edge-orchestrator.intel/v1",
				},
				ChildDeploymentList: childDeploymentList,
			},
		}
		return setInstance, nil
	} else if scenario == "update" {
		setInstance := &deploymentv1beta1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:            d.Name,
				Namespace:       d.Namespace,
				Labels:          labelList,
				ResourceVersion: resourceVersion,
			},
			Spec: deploymentv1beta1.DeploymentSpec{
				DisplayName:          d.DisplayName,
				Project:              d.Project,
				DeploymentPackageRef: dpRef,
				Applications:         applicationsList,
				DeploymentType:       deploymentType(d.DeploymentType),
				NetworkRef: corev1.ObjectReference{
					Name:       d.NetworkName,
					Kind:       "Network",
					APIVersion: "network.edge-orchestrator.intel/v1",
				},
			},
		}
		return setInstance, nil
	}

	return nil, errors.NewInvalid("cannot %s deployment", scenario)
}

// Set the details of the deployment cluster and return the instance.
func createDeploymentClusterCr(dc *deploymentv1beta1.DeploymentCluster) *deploymentpb.Cluster {
	// Create list for apps in deployment
	appList := make([]*deploymentpb.App, 0)
	if dc != nil && len(dc.Status.Apps) != 0 {
		appList = make([]*deploymentpb.App, len(dc.Status.Apps))
		for i, app := range dc.Status.Apps {
			// Create app summary
			appSummary := &deploymentpb.Summary{
				Total:   utils.ToInt32Clamped(app.Status.Summary.Total),
				Running: utils.ToInt32Clamped(app.Status.Summary.Running),
				Down:    utils.ToInt32Clamped(app.Status.Summary.Down),
				Unknown: utils.ToInt32Clamped(app.Status.Summary.Unknown),
				Type:    string(app.Status.Summary.Type),
			}

			appState := dc.Status.Apps[i].Status.State

			// Create app status
			appStatus := &deploymentpb.Deployment_Status{
				State:   deploymentState(string(appState)),
				Message: app.Status.Message,
				Summary: appSummary,
			}

			// Append to app list
			appList[i] = &deploymentpb.App{
				Id:     app.Id,
				Name:   app.Name,
				Status: appStatus,
			}
		}
	}

	summary := &deploymentpb.Summary{
		Total:   utils.ToInt32Clamped(dc.Status.Status.Summary.Total),
		Running: utils.ToInt32Clamped(dc.Status.Status.Summary.Running),
		Down:    utils.ToInt32Clamped(dc.Status.Status.Summary.Down),
		Unknown: utils.ToInt32Clamped(dc.Status.Status.Summary.Unknown),
		Type:    string(dc.Status.Status.Summary.Type),
	}

	deployState := dc.Status.Status.State

	status := &deploymentpb.Deployment_Status{
		State:   deploymentState(string(deployState)),
		Message: dc.Status.Status.Message,
		Summary: summary,
	}

	cluster := &deploymentpb.Cluster{
		Name:   dc.Status.Name,
		Id:     dc.Spec.ClusterID,
		Status: status,
		Apps:   appList,
	}

	return cluster
}

// Return the matching deployment object.
func matchUIDDeployment(ctx context.Context, UID string, namespace string, s *DeploymentSvc, listOpts metav1.ListOptions) (*deploymentv1beta1.Deployment, error) {
	var deployment deploymentv1beta1.Deployment

	// ListOptions filters with LabelSelector and FieldSelector only. FieldSelector only
	// supports metadata.namespace or metadata.name
	// get all deployments
	deployments, err := s.crClient.Deployments(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, k8serrors.K8sToTypedError(err)
	}

	// loop thru all deployments and compare the UID
	// once matched then return object
	for _, val := range deployments.Items {
		if string(val.ObjectMeta.UID) == UID {
			deployment = val
			break
		}
	}

	return &deployment, nil
}

// Create all secrets.
func createSecrets(ctx context.Context, k8sClient *kubernetes.Clientset, d *Deployment) (*Deployment, error) {
	overrideValues := map[string]string{}
	overrideValuesMasked := map[string]string{}
	var contents string
	var contentsMasked string

	rsRepoUpdated := false
	rsRepoURL := os.Getenv("RS_PROXY_REPO")
	rsRepoSec := os.Getenv("RS_PROXY_REPO_SECRET")
	rsRemoteNs := os.Getenv("RS_PROXY_REMOTE_NS")

	// Get the values from user input OverrideValues - these should already have masked values replaced
	// with real values from the UnmaskSecrets call in UpdateDeployment
	for _, value := range d.OverrideValues {
		notRawMsg, err := json.Marshal(value.Values)
		if err == nil {
			contents = string(notRawMsg)
		}
		overrideValues[value.AppName] = contents
	}

	// Get the values including masked secrets from user input OverrideValuesMasked
	for _, value := range d.OverrideValuesMasked {
		notRawMsg, err := json.Marshal(value.Values)
		if err == nil {
			contentsMasked = string(notRawMsg)
		}
		overrideValuesMasked[value.AppName] = contentsMasked
	}

	// Create new Secrets
	d.ProfileSecretName = make(map[string]string)
	d.ValueSecretName = make(map[string]string)
	d.RepoSecretName = make(map[string]string)
	d.ImageRegistrySecretName = make(map[string]string)

	for _, app := range *d.HelmApps {
		contents = ""
		// ProfileSecretName contains the override values at run-time per application.
		secretName := fmt.Sprintf("%s-%s-%s-profile", d.Name, app.Name, strings.ToLower(app.Version))
		d.ProfileSecretName[app.Name] = secretName

		data := map[string]string{}
		data["values"] = app.Values

		err := utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
		if err != nil {
			utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
			return d, err
		}

		if overrides, ok := overrideValues[app.Name]; ok {
			val, err := yaml2.JSONToYAML([]byte(overrides))
			if err != nil {
				contents = "# WARNING: Values are not valid JSON, could not convert to YAML\n" + overrides + "\n"
			} else {
				contents = string(val)
			}
		}

		// ValueSecretName contains the override values at run-time per application.
		// If no overrides provided, don't create secret to avoid empty data secret.
		if contents != "" {
			secretName := fmt.Sprintf("%s-%s-%s-overrides", d.Name, app.Name, strings.ToLower(app.Version))
			d.ValueSecretName[app.Name] = secretName

			data := map[string]string{}
			data["values"] = contents

			err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
			if err != nil {
				utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}
		}

		// RepoSecretName contains the auth secret for a private helm repository.
		// Valid only when Repo is provided.
		if app.Repo != "" {
			data := map[string]string{}
			data["cacerts"] = app.HelmCredential.CaCerts
			data["password"] = app.HelmCredential.Password
			data["username"] = app.HelmCredential.Username

			if (data["cacerts"] != "") || ((data["password"] != "") && (data["username"] != "")) {
				secretName := fmt.Sprintf("%s-%s-%s-helmrepo", d.Name, app.Name, strings.ToLower(app.Version))
				d.RepoSecretName[app.Name] = secretName
				err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
				if err != nil {
					utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
					return d, err
				}
			}
		}

		// ImageRegistrySecretName contains the auth secret for the private image
		// registry. Valid only when ImageRegistry is provided.
		if app.ImageRegistry != "" {
			data := map[string]string{}
			data["password"] = app.DockerCredential.Password
			data["username"] = app.DockerCredential.Username

			if (data["password"] != "") && (data["username"] != "") {
				secretName := fmt.Sprintf("%s-%s-%s-imagerepo", d.Name, app.Name, strings.ToLower(app.Version))
				d.ImageRegistrySecretName[app.Name] = secretName
				err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
				if err != nil {
					utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
					return d, err
				}
			}
		}

		// Handle parameter template secrets - these contain the real secret values
		if d.ParameterTemplateSecrets[app.Name] != "" {
			secretName := fmt.Sprintf("%s-%s-%s-secret", d.Name, app.Name, d.ProfileName)

			data := map[string]string{}
			data["values"] = d.ParameterTemplateSecrets[app.Name]
			err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
			if err != nil {
				utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}

			// Create masked override secret for UI display
			if overrides, ok := overrideValuesMasked[app.Name]; ok {
				val, err := yaml2.JSONToYAML([]byte(overrides))
				if err != nil {
					contentsMasked = "# WARNING: Values are not valid JSON, could not convert to YAML\n" + overrides + "\n"
				} else {
					contentsMasked = string(val)
				}
			}

			// contentsMasked contains the override values masked at run-time per application.
			// This secret will be used when GetDeployment is called to avoid showing secrets from override values.
			if contentsMasked != "" {
				secretName := fmt.Sprintf("%s-%s-%s-overrides-masked", d.Name, app.Name, strings.ToLower(app.Version))

				data := map[string]string{}
				data["values"] = contentsMasked

				err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
				if err != nil {
					utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
					return d, err
				}
			}
		}

		if app.Repo != "" && app.Repo == rsRepoURL && !rsRepoUpdated {
			err := configureRsProxy(ctx, k8sClient, d.Namespace, rsRepoSec, rsRemoteNs)
			if err != nil {
				utils.LogActivity(ctx, "create", "ADM", fmt.Sprintf("%v", err))
				return d, err
			}
			rsRepoUpdated = true
		}
	}

	return d, nil
}

// Update secret with OwnerReference.
func updateOwnerRefSecret(ctx context.Context, kc *kubernetes.Clientset, ownerReferenceList []metav1.OwnerReference, secretName string, namespace string) error {
	secret, err := kc.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
		return k8serrors.K8sToTypedError(err)
	}

	secret.ObjectMeta.OwnerReferences = ownerReferenceList

	_, err = kc.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
		return k8serrors.K8sToTypedError(err)
	}

	return nil
}

func ownerReferenceList(d *Deployment, kindType string) []metav1.OwnerReference {
	var ownerReferenceList []metav1.OwnerReference
	apiVersion := fmt.Sprintf("%s/%s", deploymentv1beta1.GroupVersion.Group, deploymentv1beta1.GroupVersion.Version)
	useController := true

	ownerReference := metav1.OwnerReference{
		Name:       d.Name,
		Kind:       kindType,
		APIVersion: apiVersion,
		UID:        types.UID(d.DeployID),
		Controller: &useController,
	}

	ownerReferenceList = append(ownerReferenceList, ownerReference)
	return ownerReferenceList
}

// Update all secrets with Owner reference. This needs to happen after deployment is created since deployment UID is required.
func updateOwnerRefSecrets(ctx context.Context, k8sClient *kubernetes.Clientset, d *Deployment) (*Deployment, error) {
	ownerReferenceList := ownerReferenceList(d, "Deployment")

	for _, app := range *d.HelmApps {
		secretName := d.ProfileSecretName[app.Name]

		err := updateOwnerRefSecret(ctx, k8sClient, ownerReferenceList, secretName, d.Namespace)
		if err != nil {
			utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
			return d, err
		}

		if d.ValueSecretName[app.Name] != "" {
			secretName := d.ValueSecretName[app.Name]

			err = updateOwnerRefSecret(ctx, k8sClient, ownerReferenceList, secretName, d.Namespace)
			if err != nil {
				utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}

			if d.ParameterTemplateSecrets[app.Name] != "" {
				// Update for masked override values
				err = updateOwnerRefSecret(ctx, k8sClient, ownerReferenceList, secretName+"-masked", d.Namespace)
				if err != nil {
					if !(errors.IsNotFound(err)) {
						utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
						return d, err
					}
				}
			}
		}

		if d.RepoSecretName[app.Name] != "" {
			secretName := d.RepoSecretName[app.Name]

			err = updateOwnerRefSecret(ctx, k8sClient, ownerReferenceList, secretName, d.Namespace)
			if err != nil {
				utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}
		}

		if d.ImageRegistrySecretName[app.Name] != "" {
			secretName := d.ImageRegistrySecretName[app.Name]

			err = updateOwnerRefSecret(ctx, k8sClient, ownerReferenceList, secretName, d.Namespace)
			if err != nil {
				utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}
		}

		if d.ParameterTemplateSecrets[app.Name] != "" {
			secretName := fmt.Sprintf("%s-%s-%s-secret", d.Name, app.Name, d.ProfileName)

			err = updateOwnerRefSecret(ctx, k8sClient, ownerReferenceList, secretName, d.Namespace)
			if err != nil {
				utils.LogActivity(ctx, "update", "ADM", "cannot update secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}
		}
	}

	return d, nil
}

// Delete all secrets.
func deleteSecrets(ctx context.Context, k8sClient *kubernetes.Clientset, deployment *deploymentv1beta1.Deployment) error {
	var secretName string
	namespace := deployment.ObjectMeta.Namespace

	for _, app := range deployment.Spec.Applications {
		secretName = app.ProfileSecretName
		err := utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
		if errors.IsNotFound(err) {
			log.Warnf("Secret %s not found, continuing", secretName)
		} else if err != nil {
			return err
		}

		secretName = app.ValueSecretName
		if secretName != "" {
			err := utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
			if errors.IsNotFound(err) {
				log.Warnf("Secret %s not found, continuing", secretName)
			} else if err != nil {
				return err
			}

			// Delete masked override values
			err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName+"-masked")
			if errors.IsNotFound(err) {
				log.Warnf("Secret %s not found, continuing", secretName)
			} else if err != nil {
				return err
			}
		}

		secretName = app.HelmApp.RepoSecretName
		if secretName != "" && secretName != os.Getenv("RS_PROXY_REPO_SECRET") {
			err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
			if errors.IsNotFound(err) {
				log.Warnf("Secret %s not found, continuing", secretName)
			} else if err != nil {
				return err
			}
		}

		secretName = app.HelmApp.ImageRegistrySecretName
		if secretName != "" {
			err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
			if errors.IsNotFound(err) {
				log.Warnf("Secret %s not found, continuing", secretName)
			} else if err != nil {
				return err
			}
		}

		secretName = fmt.Sprintf("%s-%s-%s-secret", deployment.Name, app.Name, deployment.Spec.DeploymentPackageRef.ProfileName)
		err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
		if errors.IsNotFound(err) {
			log.Warnf("Secret %s not found, continuing", secretName)
		} else if err != nil {
			return err
		}

	}

	return nil
}

func removeIndex(s []string, index int) []string {
	// This function returns a new slice with the element at 'index' removed, without modifying the original slice.
	result := make([]string, 0, len(s)-1)
	result = append(result, s[:index]...)
	result = append(result, s[index+1:]...)
	return result
}

func updatePbValue(s *structpb.Struct, structKeys []string, inValInt int, inValStr string, inValType string, currentDepth int) (int, string) {
	// Check if the current depth exceeds the maximum depth
	if currentDepth > maxDepth {
		fmt.Println("Maximum recursion depth reached")
		return inValInt, inValStr
	}

	for k := range s.Fields {
		if k != structKeys[0] {
			continue
		}

		v := s.Fields[k]
		if v.Kind == nil {
			continue
		}

		switch v.Kind.(type) {
		case *structpb.Value_StringValue:
			if inValType == "number" {
				inValInt, _ = strconv.Atoi(v.GetStringValue())
				s.Fields[k] = structpb.NewNumberValue(float64(inValInt))
			} else if inValType == "boolean" {
				inValStr = v.GetStringValue()
				var boolVal bool
				if inValStr == "true" {
					boolVal = true
				} else {
					boolVal = false
				}
				s.Fields[k] = structpb.NewBoolValue(boolVal)
			}
		case *structpb.Value_StructValue:
			inValInt, inValStr = updatePbValue(v.GetStructValue(), removeIndex(structKeys, 0), inValInt, inValStr, inValType, currentDepth+1)
		}
	}

	return inValInt, inValStr
}

// Mask secret values in the structpb.Struct. Modifies the function argument s in place.
// Returns the original secret value.
func maskPbValue(s *structpb.Struct, structKeys []string, inValStr string, currentDepth int) string {
	// Check if the current depth exceeds the maximum depth
	if currentDepth > maxDepth {
		fmt.Println("Maximum recursion depth reached")
		return inValStr
	}

	for k := range s.Fields {
		if k != structKeys[0] {
			continue
		}

		v := s.Fields[k]
		if v.Kind == nil {
			continue
		}

		switch v.Kind.(type) {
		case *structpb.Value_StringValue:
			inValStr = v.GetStringValue()

			s.Fields[k] = structpb.NewStringValue(MaskedValuePlaceholder)
		case *structpb.Value_StructValue:
			inValStr = maskPbValue(v.GetStructValue(), removeIndex(structKeys, 0), inValStr, currentDepth+1)
		}
	}

	return inValStr
}

// Remove all items in list2 from list1 and return the result.
func cleanUpList(list1, list2 []string) []string {
	outList := []string{}
	for _, item1 := range list1 {
		found := false
		for _, item2 := range list2 {
			if item1 == item2 {
				found = true
				break
			}
		}
		if !found {
			outList = append(outList, item1)
		}
	}
	return outList
}

// Get all keys in the structpb.Struct including nested keys.
func getAllPbStructKeys(s *structpb.Struct, emptyValKeys []string, currentDepth int) ([]string, []string) {
	var keys []string
	// Check if the current depth exceeds the maximum depth
	if currentDepth > maxDepth {
		fmt.Println("Maximum recursion depth reached")
		return keys, emptyValKeys
	}

	for k := range s.Fields {
		keys = append(keys, k)

		v := s.Fields[k]
		if v.Kind == nil {
			continue
		}

		switch v.Kind.(type) {
		case *structpb.Value_StringValue:
			// If value is empty then add to emptyValKeys list and delete key
			if v.GetStringValue() == "" {
				emptyValKeys = append(emptyValKeys, k)
				delete(s.Fields, k)
			}
		case *structpb.Value_StructValue:
			nestedKeys, emptyValKeys := getAllPbStructKeys(v.GetStructValue(), emptyValKeys, currentDepth+1)

			// Remove key from allOverrideKeys list if value is empty
			nestedKeys = cleanUpList(nestedKeys, emptyValKeys)

			for _, nk := range nestedKeys {
				keys = append(keys, k+"."+nk)
			}
		}
	}

	return keys, emptyValKeys
}

// Trim spaces from all string values in the structpb.Struct. Moddifies the function argument in place.
func trimAllPbStructStrings(s *structpb.Struct, currentDepth int) {
	// Check if the current depth exceeds the maximum depth
	if currentDepth > maxDepth {
		fmt.Println("Maximum recursion depth reached")
		return
	}

	for k := range s.Fields {
		v := s.Fields[k]
		if v.Kind == nil {
			continue
		}

		switch v.Kind.(type) {
		case *structpb.Value_StringValue:
			if strVal := v.GetStringValue(); strVal != "" {
				trimmed := strings.TrimSpace(strVal)
				if trimmed != strVal {
					s.Fields[k] = structpb.NewStringValue(trimmed)
				}
			}
		case *structpb.Value_StructValue:
			trimAllPbStructStrings(v.GetStructValue(), currentDepth+1)
		}
	}
}

// Check parameter template values for mandatory and secret keys.
// Masks the secrets and puts the original secrets into ParameterTemplateSecrets map.
// Returns error if any mandatory values are missing.
func checkParameterTemplate(d *Deployment, allOverrideKeys map[string][]string) (*Deployment, error) {
	var notFoundApp []string
	var OverrideValuesNotMasked []*deploymentpb.OverrideValues
	d.ParameterTemplateSecrets = make(map[string]string)

	// Make copy of original OverrideValues values with masking
	// to use for GetDeployments to mask any secrets
	d.OverrideValuesMasked = d.OverrideValues

	// Make copy of original OverrideValues values without masking
	// to add to fleet's overrides.yaml
	for _, oVal := range d.OverrideValues {
		notMaskedValues := proto.Clone(oVal.Values).(*structpb.Struct)
		OverrideValuesNotMasked = append(OverrideValuesNotMasked, &deploymentpb.OverrideValues{
			AppName:         oVal.AppName,
			TargetNamespace: oVal.TargetNamespace,
			Values:          notMaskedValues,
		})
	}
	d.OverrideValues = OverrideValuesNotMasked

	for _, app := range *d.HelmApps {
		addedToNotFoundApp := false
		appSecretVals := make(map[string]string)

		// Validate parameter template and create secret if needed
		for _, val := range app.ParameterTemplates {
			// Convert to correct value type
			for _, k := range allOverrideKeys[app.Name] {
				if k == val.Name {
					var inValInt int
					var inValStr string
					for _, oVal := range d.OverrideValues {
						if oVal.AppName == app.Name {
							keys := strings.Split(val.Name, ".")
							_, _ = updatePbValue(oVal.Values, keys, inValInt, inValStr, val.Type, 0)
						}
					}
				}
			}

			foundMandatory := false
			// Search in override values input to confirm mandatory value was provided
			if val.Mandatory {
				for _, k := range allOverrideKeys[app.Name] {
					if k == val.Name {
						foundMandatory = true
					}
				}
				// Mandatory value not found in override values so append which app has missing value
				// Prevent duplicate in notFoundApp slice by setting addedToNotFoundApp
				if !(foundMandatory) && !(addedToNotFoundApp) {
					addedToNotFoundApp = true
					notFoundApp = append(notFoundApp, app.Name)
				}
			}

			// Search in override values input to confirm mandatory value was provided
			if val.Secret {
				for _, k := range allOverrideKeys[app.Name] {
					if k == val.Name {
						var secretVal string
						for _, oVal := range d.OverrideValuesMasked {
							// Make sure values belong to same app
							// ie 2 apps can share same key
							if oVal.AppName == app.Name {
								// will update new values with masking to output to user
								secretVal = maskPbValue(oVal.Values, strings.Split(val.Name, "."), secretVal, 0)
								appSecretVals[val.Name] = secretVal
							}
						}
					}
				}
			}
		}

		// Only set ParameterTemplateSecrets if secrets found
		if len(appSecretVals) > 0 {
			jsonAppSecretVals, err := json.Marshal(appSecretVals)
			if err != nil {
				return d, errors.NewInvalid("Error: %v", err)
			}
			d.ParameterTemplateSecrets[app.Name] = string(jsonAppSecretVals)
		}
	}

	// Error if any mandatory value is missing
	if len(notFoundApp) > 0 {
		msg := fmt.Sprintf("application %v is missing mandatory override profile values", notFoundApp[0])
		if len(notFoundApp) > 1 {
			var notFoundAppStr string
			for _, v := range notFoundApp {
				notFoundAppStr = v + ", " + notFoundAppStr
			}
			notFoundAppStr = strings.TrimSuffix(notFoundAppStr, ", ")
			msg = fmt.Sprintf("applications %s are missing mandatory override profile values", notFoundAppStr)
		}
		return d, errors.NewInvalid(msg)
	}

	return d, nil
}

func mergeAllAppTargetClusters(ctx context.Context, d *Deployment) error {
	if len(d.TargetClusters) == 0 {
		// In case when there is no targetClusters, add an entry for all apps
		// in targetcluster for each label/clusterId in AllAppTargetCluster
		if len(*d.HelmApps) != 0 {
			for _, app := range *d.HelmApps {
				target := d.AllAppTargetClusters
				targetClusterItem := &deploymentpb.TargetClusters{
					AppName: app.Name,
				}
				if d.DeploymentType == string(deploymentv1beta1.Targeted) {
					targetClusterItem.ClusterId = target.ClusterId
				} else if d.DeploymentType == string(deploymentv1beta1.AutoScaling) {
					targetClusterItem.Labels = make(map[string]string, 0)
					targetClusterItem.Labels = target.Labels
				}
				d.TargetClusters = append(d.TargetClusters, targetClusterItem)
			}
		}
	} else {
		// In case targetClusters field exists, then check if AllAppTargetClusters
		// should be added as a NEW separate target for each app
		// We add it as a new target if it's NOT an exact duplicate of any existing target
		if len(*d.HelmApps) != 0 {
			for _, app := range *d.HelmApps {
				target := d.AllAppTargetClusters

				// Check if this exact target already exists for this app
				targetExists := false
				for _, tc := range d.TargetClusters {
					if tc.AppName == app.Name {
						// Check for exact match (all key-value pairs must match)
						allLabelsMatch := true
						if len(tc.Labels) != len(target.Labels) {
							allLabelsMatch = false
						} else {
							for key, val := range target.Labels {
								if tc.Labels[key] != val {
									allLabelsMatch = false
									break
								}
							}
						}
						if allLabelsMatch {
							targetExists = true
							utils.LogActivity(ctx, "create", "ADM", "Target already exists for app "+app.Name)
							break
						}
					}
				}

				// If this exact target doesn't exist, add it as a NEW target
				if !targetExists {
					targetClusterItem := &deploymentpb.TargetClusters{
						AppName: app.Name,
					}
					if d.DeploymentType == string(deploymentv1beta1.Targeted) {
						targetClusterItem.ClusterId = target.ClusterId
					} else if d.DeploymentType == string(deploymentv1beta1.AutoScaling) {
						targetClusterItem.Labels = make(map[string]string)
						for k, v := range target.Labels {
							targetClusterItem.Labels[k] = v
						}
					}
					d.TargetClusters = append(d.TargetClusters, targetClusterItem)
					utils.LogActivity(ctx, "create", "ADM", "Added new target for app "+app.Name)
				}
			}
		}
	}

	return nil
}

// Existing secret name containing the private git repo access credentials.
// The secret needs to be in the same namespace the GitRepo CR is in.
func createGitClientSecret(ctx context.Context, k8sClient *kubernetes.Clientset, d *Deployment) error {
	secretName := deploymentv1beta1.FleetGitSecretName

	// Check if the secret already exists to avoid overhead early on
	value, err := utils.GetSecretValue(ctx, k8sClient, d.Namespace, secretName)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
			return err
		}
	}

	// Secret already exists, no need to create new vault manager
	if value != nil {
		return nil
	}

	secretServiceEnabled, err := utils.IsSecretServiceEnabled()
	if err != nil {
		return err
	}

	if !(secretServiceEnabled) {
		log.Infof("Secret service disabled, cannot create %s secret", secretName)
		return nil
	}

	data := map[string]string{}

	vaultManager := vault.NewManager(utils.GetSecretServiceEndpoint(), utils.GetServiceAccount(), utils.GetSecretServiceMount())
	vaultClient, err := vaultManager.GetVaultClient(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err := vaultManager.Logout(ctx, vaultClient); err != nil {
			log.Errorf("failed to logout from vault: %v", err)
		}
	}()

	user, err := vaultManager.GetSecretValueString(ctx, vaultClient, utils.GetSecretServiceGitServicePath(), utils.GetSecretServiceGitServiceKVKeyUsername())
	if err != nil {
		return err
	}

	password, err := vaultManager.GetSecretValueString(ctx, vaultClient, utils.GetSecretServiceGitServicePath(), utils.GetSecretServiceGitServiceKVKeyPassword())
	if err != nil {
		return err
	}

	data["password"] = password
	data["username"] = user

	err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, true)
	if err != nil {
		utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
		return err
	}

	log.Infof("Created %s secret in %s namespace", secretName, d.Namespace)
	return nil
}

func configureRsProxy(ctx context.Context, s *kubernetes.Clientset, nsName string, rbName string, remoteNsName string) error {
	err := utils.UpdateNsLabels(ctx, s, nsName)
	if err != nil {
		utils.LogActivity(ctx, "create", "ADM", "cannot update labels on namespace "+nsName+" "+fmt.Sprintf("%v", err))
		return err
	}

	err = utils.CreateRoleBinding(ctx, s, nsName, rbName, remoteNsName)
	if err != nil {
		utils.LogActivity(ctx, "create", "ADM", "cannot create rolebinding "+rbName+" "+fmt.Sprintf("%v", err))
		return err
	}

	// Create rolebinding for fleet-default namespace since Fleet creates external secrets there
	fleetDefaultNs := "fleet-default"
	err = utils.CreateRoleBinding(ctx, s, fleetDefaultNs, rbName, remoteNsName)
	if err != nil {
		utils.LogActivity(ctx, "create", "ADM", "cannot create rolebinding "+rbName+" for fleet-default namespace "+fmt.Sprintf("%v", err))
		return err
	}

	return nil
}

func UnmaskSecrets(current *structpb.Struct, previous *structpb.Struct, prefix string) {
	for k, v := range current.Fields {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		switch val := v.Kind.(type) {
		case *structpb.Value_StringValue:
			if val.StringValue == MaskedValuePlaceholder {
				if prevVal, ok := previous.Fields[fullKey]; ok {
					if prevStr, ok := prevVal.Kind.(*structpb.Value_StringValue); ok {
						current.Fields[k] = structpb.NewStringValue(prevStr.StringValue)
					}
				}
			}
		case *structpb.Value_StructValue:
			UnmaskSecrets(val.StructValue, previous, fullKey)
		case *structpb.Value_ListValue:
			for i, item := range val.ListValue.Values {
				if itemStruct, ok := item.Kind.(*structpb.Value_StructValue); ok {
					UnmaskSecrets(itemStruct.StructValue, previous, fullKey+fmt.Sprintf("[%d]", i))
				}
			}
		}
	}
}
