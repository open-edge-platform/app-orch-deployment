// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	"context"
	"fmt"
	clientv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/appdeploymentclient/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/k8serrors"

	"github.com/open-edge-platform/orch-library/go/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	catalog "github.com/open-edge-platform/app-orch-catalog/pkg/api/catalog/v3"
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	deploymentv1beta1 "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/v1beta1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/internal/catalogclient"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils"
)

// Create api extensions.
func createAPIExtCrs(ctx context.Context, s *DeploymentSvc, d *Deployment) error {
	apiExtEnabled, err := utils.GetAPIExtEnabled()
	if err != nil {
		return err
	}

	// Don't create CR if extension disabled.
	if apiExtEnabled == "false" {
		return nil
	}

	apie, err := catalogclient.CatalogLookupAPIExtensions(ctx, s.catalogClient, d.AppName, d.AppVersion)
	if err != nil {
		return err
	}

	// Don't create CR if no extension required.
	if len(apie) == 0 {
		return nil
	}

	endpoints := func(e *catalog.APIExtension) []deploymentv1beta1.ProxyEndpoint {
		result := []deploymentv1beta1.ProxyEndpoint{}
		for _, ep := range e.GetEndpoints() {
			result = append(result, deploymentv1beta1.ProxyEndpoint{
				ServiceName: ep.ServiceName,
				Path:        ep.ExternalPath,
				Backend:     ep.InternalPath,
				Scheme:      ep.Scheme,
				AuthType:    ep.AuthType,
				AppName:     ep.AppName,
			})
		}
		return result
	}

	uiExtensions := func(e *catalog.APIExtension) []deploymentv1beta1.UIExtension {
		result := []deploymentv1beta1.UIExtension{}
		if e.UiExtension != nil && len(e.UiExtension.ServiceName) > 0 {
			result = append(result, deploymentv1beta1.UIExtension{
				ServiceName: e.UiExtension.ServiceName,
				Description: e.UiExtension.Description,
				Label:       e.UiExtension.Label,
				FileName:    e.UiExtension.FileName,
				AppName:     e.UiExtension.AppName,
				ModuleName:  e.UiExtension.ModuleName,
			})
		}
		return result
	}

	targetsList := map[string]string{}
	for _, target := range d.TargetClusters {
		if d.AppName == target.AppName {
			if d.DeploymentType == string(deploymentv1beta1.Targeted) {
				targetsList[string(deploymentv1beta1.ClusterName)] = target.ClusterId
			} else if d.DeploymentType == string(deploymentv1beta1.AutoScaling) {
				targetsList = target.Labels
			}
			break
		}
	}

	var e *deploymentv1beta1.APIExtension

	// Create APIExtension CR for each extension if it does not exist
	for _, ext := range apie {
		crName := fmt.Sprintf("%s-%s-ae", d.Name, ext.Name)

		_, err := s.crClient.APIExtensions(d.Namespace).Get(ctx, crName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return k8serrors.K8sToTypedError(err)
			}
		}

		// Currently, only 1 api-ext per cluster is supported
		// So name uniqueness is not a requirement
		apiGroup := deploymentv1beta1.APIGroup{
			Name:    ext.Name,
			Version: ext.Version,
		}

		// APIExtension does not exist, create CR
		// Add deployId Label to map all CRs to deployment
		e = &deploymentv1beta1.APIExtension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      crName,
				Namespace: d.Namespace,
				Labels: map[string]string{
					string(deploymentv1beta1.DeploymentID): d.DeployID,
				},
			},
			Spec: deploymentv1beta1.APIExtensionSpec{
				DisplayName:        ext.DisplayName,
				Project:            d.Project,
				APIGroup:           apiGroup,
				ProxyEndpoints:     endpoints(ext),
				AgentClusterLabels: targetsList,
				UIExtensions:       uiExtensions(ext),
			},
		}

		_, err = s.crClient.APIExtensions(d.Namespace).Create(ctx, e, metav1.CreateOptions{})
		if err != nil {
			return k8serrors.K8sToTypedError(err)
		}
	}

	return nil
}

// Delete the api extension associated to the deployment.
func deleteAPIExtCrs(ctx context.Context, crClient clientv1beta1.AppDeploymentClientInterface, d *Deployment) error {
	apiExtEnabled, err := utils.GetAPIExtEnabled()
	if err != nil {
		return err
	}

	// Don't delete CR if extension disabled.
	if apiExtEnabled == "false" {
		return nil
	}

	labelSelector := metav1.LabelSelector{
		MatchLabels: map[string]string{string(deploymentv1beta1.DeploymentID): d.DeployID},
	}

	listOpts := metav1.ListOptions{
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),
	}

	// List all api extension objects and match the deployment-id of deployment
	apiExtensions, err := crClient.APIExtensions("").List(ctx, listOpts)
	if err != nil {
		return k8serrors.K8sToTypedError(err)
	}

	for _, ext := range apiExtensions.Items {
		// Delete
		err = crClient.APIExtensions(d.Namespace).Delete(ctx, ext.ObjectMeta.Name, metav1.DeleteOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return k8serrors.K8sToTypedError(err)
		}
	}

	return nil
}

func (s *DeploymentSvc) GetAPIExtension(ctx context.Context, in *deploymentpb.GetAPIExtensionRequest) (*deploymentpb.GetAPIExtensionResponse, error) {
	if in == nil || in.Name == "" {
		return nil, errors.Status(errors.NewInvalid("incomplete request")).Err()
	}

	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		return nil, errors.Status(errors.NewForbidden("cannot get API extensions", err)).Err()
	}

	apiExtEnabled, err := utils.GetAPIExtEnabled()
	if err != nil {
		return nil, errors.Status(err).Err()
	}

	if apiExtEnabled == "false" {
		return nil, errors.Status(errors.NewNotFound("cannot get API extension: extensions are disabled")).Err()
	}

	namespace, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	name := in.Name
	// get APIExtension deployment object by name
	apiExtension, err := s.crClient.APIExtensions(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	token := apiExtension.Status.TokenSecretRef.GeneratedToken

	utils.LogActivity(ctx, "get", "ADM", "APIExtension "+name)
	return &deploymentpb.GetAPIExtensionResponse{
		ApiExtension: &deploymentpb.APIExtension{
			Name:  name,
			Token: token,
		},
	}, nil
}

func (s *DeploymentSvc) ListUIExtensions(ctx context.Context, in *deploymentpb.ListUIExtensionsRequest) (*deploymentpb.ListUIExtensionsResponse, error) {
	if err := s.protoValidator.Validate(in); err != nil {
		log.Warnf("%v", err)
		return nil, errors.Status(errors.NewInvalid("%v", err)).Err()
	}

	// RBAC auth
	if err := s.AuthCheckAllowed(ctx, in); err != nil {
		return nil, errors.Status(errors.NewForbidden("cannot list UI extensions ", err)).Err()
	}

	apiExtEnabled, err := utils.GetAPIExtEnabled()
	if err != nil {
		return nil, errors.Status(err).Err()
	}

	if apiExtEnabled == "false" {
		return nil, errors.Status(errors.NewNotFound("cannot list UI extensions: extensions are disabled")).Err()
	}

	namespace, err := s.GetActiveProjectID(ctx)
	if err != nil {
		msg := fmt.Sprintf("failed to get tenant project ID %s", err.Error())
		return nil, errors.Status(errors.NewUnavailable(msg)).Err()
	}

	// get APIExtension deployment object by name
	apiExtensions, err := s.crClient.APIExtensions(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Status(k8serrors.K8sToTypedError(err)).Err()
	}

	var uiExtensions []*deploymentpb.UIExtension
	for index, apiExtension := range apiExtensions.Items {
		if len(apiExtension.Spec.UIExtensions) > 0 {
			u := apiExtension.Spec.UIExtensions[index]
			uiExtensions = append(uiExtensions, &deploymentpb.UIExtension{
				ServiceName: u.ServiceName,
				Description: u.Description,
				Label:       u.Label,
				FileName:    u.FileName,
				AppName:     u.AppName,
				ModuleName:  u.ModuleName})
		}
	}

	utils.LogActivity(ctx, "list", "ADM", "UIExtensions")
	return &deploymentpb.ListUIExtensionsResponse{
		UiExtensions: uiExtensions,
	}, nil
}
