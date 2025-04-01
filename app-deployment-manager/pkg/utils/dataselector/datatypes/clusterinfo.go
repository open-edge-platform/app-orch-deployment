// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
)

type ClusterInfo struct {
	clusterInfo *deploymentpb.ClusterInfo
}

func (ci *ClusterInfo) GetField(name dataselector.FieldName) dataselector.Comparable {
	switch name {
	case "name":
		return dataselector.StdComparableString(ci.clusterInfo.Name)
	case "id":
		return dataselector.StdComparableString(ci.clusterInfo.Id)
	default:
		return nil
	}
}

func ToDataItemsFromClusterInfoList(clusterInfoList []*deploymentpb.ClusterInfo) []dataselector.DataItem {
	clusterInfoDataItems := make([]dataselector.DataItem, len(clusterInfoList))
	for i, clusterInfo := range clusterInfoList {
		clusterInfoDataItem := &ClusterInfo{
			clusterInfo: clusterInfo,
		}
		clusterInfoDataItems[i] = clusterInfoDataItem
	}
	return clusterInfoDataItems
}

func FromDataItemsToClusterInfoList(dataItems []dataselector.DataItem) []*deploymentpb.ClusterInfo {
	clusterInfoList := make([]*deploymentpb.ClusterInfo, len(dataItems))
	for i, dataItem := range dataItems {
		clusterInfoDataItem := dataItem.(*ClusterInfo)
		clusterInfoList[i] = clusterInfoDataItem.clusterInfo
	}

	return clusterInfoList
}
