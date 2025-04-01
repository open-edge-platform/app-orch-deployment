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

// Set the details of the deployment and return the instance.
func createDeploymentCr(d *Deployment, scenario string, resourceVersion string) (*deploymentv1beta1.Deployment, error) {
	labelList := map[string]string{
		"app.kubernetes.io/name":       "deployment",
		"app.kubernetes.io/instance":   d.Name,
		"app.kubernetes.io/part-of":    "app-deployment-manager",
		"app.kubernetes.io/managed-by": "kustomize",
		"app.kubernetes.io/created-by": "app-deployment-manager",
	}

	activeProjectIDKey := string(deploymentv1beta1.AppOrchActiveProjectID)
	// TODO: if ActiveProjectID is empty return error in the future - tenant project ID can be empty now but it should be mandatory for the future.
	if d.ActiveProjectID != "" {
		labelList[activeProjectIDKey] = d.ActiveProjectID
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

			for _, target := range d.TargetClusters {
				if app.Name == target.AppName {
					if d.DeploymentType == string(deploymentv1beta1.Targeted) {
						target.Labels[string(deploymentv1beta1.ClusterName)] = target.ClusterId
					}

					targetsList = append(targetsList, target.Labels)
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
		Publisher:                  d.PublisherName,
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

	// Get the values from user input OverrideValues
	for _, value := range d.OverrideValues {
		notRawMsg, err := json.Marshal(value.Values)
		if err == nil {
			contents = string(notRawMsg)
		}

		overrideValues[value.AppName] = contents
	}

	// Get the values including masked secrets from user input OverrideValues
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
		secretName := fmt.Sprintf("%s-%s-%s-profile", d.Name, app.Name, app.Version)
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
			secretName := fmt.Sprintf("%s-%s-%s-overrides", d.Name, app.Name, app.Version)
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
				secretName := fmt.Sprintf("%s-%s-%s-helmrepo", d.Name, app.Name, app.Version)
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
				secretName := fmt.Sprintf("%s-%s-%s-imagerepo", d.Name, app.Name, app.Version)
				d.ImageRegistrySecretName[app.Name] = secretName
				err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
				if err != nil {
					utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
					return d, err
				}
			}
		}

		if d.ParameterTemplateSecrets[app.Name] != "" {
			secretName := fmt.Sprintf("%s-%s-%s-secret", d.Name, app.Name, d.ProfileName)

			data := map[string]string{}
			data["values"] = d.ParameterTemplateSecrets[app.Name]
			err = utils.CreateSecret(ctx, k8sClient, d.Namespace, secretName, data, false)
			if err != nil {
				utils.LogActivity(ctx, "create", "ADM", "cannot create secret "+secretName+" "+fmt.Sprintf("%v", err))
				return d, err
			}

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
				secretName := fmt.Sprintf("%s-%s-%s-overrides-masked", d.Name, app.Name, app.Version)

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
		if err != nil {
			return err
		}

		secretName = app.ValueSecretName
		if secretName != "" {
			err := utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
			if err != nil {
				return err
			}

			// Delete masked override values
			err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName+"-masked")
			if err != nil {
				if !(errors.IsNotFound(err)) {
					return err
				}
			}
		}

		secretName = app.HelmApp.RepoSecretName
		if secretName != "" && secretName != os.Getenv("RS_PROXY_REPO_SECRET") {
			err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
			if err != nil {
				return err
			}
		}

		secretName = app.HelmApp.ImageRegistrySecretName
		if secretName != "" {
			err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
			if err != nil {
				return err
			}
		}

		secretName = fmt.Sprintf("%s-%s-%s-secret", deployment.Name, app.Name, deployment.Spec.DeploymentPackageRef.ProfileName)
		err = utils.DeleteSecret(ctx, k8sClient, namespace, secretName)
		if err != nil {
			if !(errors.IsNotFound(err)) {
				return err
			}
		}
	}

	return nil
}

func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
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
			removeIndex(structKeys, 0)
			inValInt, inValStr = updatePbValue(v.GetStructValue(), removeIndex(structKeys, 0), inValInt, inValStr, inValType, currentDepth+1)
		}
	}

	return inValInt, inValStr
}

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

			s.Fields[k] = structpb.NewStringValue("********")
		case *structpb.Value_StructValue:
			removeIndex(structKeys, 0)
			inValStr = maskPbValue(v.GetStructValue(), removeIndex(structKeys, 0), inValStr, currentDepth+1)
		}
	}

	return inValStr
}

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
							_, _ = updatePbValue(oVal.Values,
								strings.Split(val.Name, "."), inValInt, inValStr, val.Type, 0)
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
								secretVal = maskPbValue(oVal.Values,
									strings.Split(val.Name, "."), secretVal, 0)

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
		// In case targetClusters field exists, then loop through helm apps
		// for every app check if there is an entry in targetClusters
		// corresponding to labels in AllAppTargetClusters
		// if not found, then add the entry for that app else move on.
		if len(*d.HelmApps) != 0 {
			for _, app := range *d.HelmApps {
				target := d.AllAppTargetClusters
				for _, tc := range d.TargetClusters {
					if tc.AppName == app.Name {
						if d.DeploymentType == string(deploymentv1beta1.AutoScaling) {
							for key, val := range target.Labels {
								if _, exists := tc.Labels[key]; !exists {
									utils.LogActivity(ctx, "create", "ADM", "label added with key "+key+" and value "+val+" for app "+app.Name)
									tc.Labels[key] = val
								} else {
									utils.LogActivity(ctx, "create", "ADM", "label already exists with key "+key+" for app "+app.Name)
								}
							}
						} else if d.DeploymentType == string(deploymentv1beta1.Targeted) {
							utils.LogActivity(ctx, "create", "ADM", "Target cluster ID "+tc.ClusterId+" already exists for app "+app.Name)
						}
					}
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

	return nil
}
