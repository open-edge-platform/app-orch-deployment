// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
)

type Cluster struct {
	cluster *deploymentpb.Cluster
}

func (ci *Cluster) GetField(name dataselector.FieldName) dataselector.Comparable {
	switch name {
	case "name":
		return dataselector.StdComparableString(ci.cluster.Name)
	case "id":
		return dataselector.StdComparableString(ci.cluster.Id)
	case "status":
		return dataselector.StdComparableString(ci.cluster.Status.State.String())
	default:
		return nil
	}
}

func ToDataItemsFromClusterList(clusterList []*deploymentpb.Cluster) []dataselector.DataItem {
	clusterListDataItems := make([]dataselector.DataItem, len(clusterList))
	for i, cluster := range clusterList {
		clusterListDataItem := &Cluster{
			cluster: cluster,
		}
		clusterListDataItems[i] = clusterListDataItem
	}
	return clusterListDataItems
}

func FromDataItemsToClusterList(dataItems []dataselector.DataItem) []*deploymentpb.Cluster {
	clusterList := make([]*deploymentpb.Cluster, len(dataItems))
	for i, dataItem := range dataItems {
		clusterListDataItem := dataItem.(*Cluster)
		clusterList[i] = clusterListDataItem.cluster
	}

	return clusterList
}
