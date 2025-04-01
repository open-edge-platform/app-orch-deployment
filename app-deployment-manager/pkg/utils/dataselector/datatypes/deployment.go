// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
)

type Deployment struct {
	deployment *deploymentpb.Deployment
}

func (dc *Deployment) GetField(name dataselector.FieldName) dataselector.Comparable {
	switch name {
	case "name":
		return dataselector.StdComparableString(dc.deployment.Name)
	case "displayName":
		return dataselector.StdComparableString(dc.deployment.DisplayName)
	case "appName":
		return dataselector.StdComparableString(dc.deployment.AppName)
	case "appVersion":
		return dataselector.StdComparableString(dc.deployment.AppVersion)
	case "deployId":
		return dataselector.StdComparableString(dc.deployment.DeployId)
	case "status":
		return dataselector.StdComparableString(dc.deployment.Status.State.String())
	default:
		return nil
	}
}

func ToDataItemsFromDeployments(deploymentList []*deploymentpb.Deployment) []dataselector.DataItem {
	dataItems := make([]dataselector.DataItem, len(deploymentList))
	for i, deployment := range deploymentList {
		deploymentDataItem := &Deployment{
			deployment: deployment,
		}
		dataItems[i] = deploymentDataItem
	}
	return dataItems
}

func FromDataItemsToDeployments(dataItems []dataselector.DataItem) []*deploymentpb.Deployment {
	deploymentList := make([]*deploymentpb.Deployment, len(dataItems))
	for i, dataItem := range dataItems {
		deploymentDataItem := dataItem.(*Deployment)
		deploymentList[i] = deploymentDataItem.deployment
	}

	return deploymentList
}
