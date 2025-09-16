// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package northbound

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector/datatypes"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/parser"
)

// converting pagination parameters from int32 to uint32
func convertPaginationParams(pageSize, offset int32) (uint32, uint32) {
	// Protobuf validation guarantees pageSize >= 0 and offset >= 0
	//nolint:gosec // G115: Values guaranteed non-negative by prior protobuf validation
	return uint32(pageSize), uint32(offset)
}

func selectDeployments(in *deploymentpb.ListDeploymentsRequest, deployments []*deploymentpb.Deployment) ([]*deploymentpb.Deployment, error) {
	var orderByList []dataselector.OrderBy

	pageSize, offset := convertPaginationParams(in.PageSize, in.Offset)
	paginationQuery := newPaginationQuery(pageSize, offset)

	err := paginationQuery.IsValidPagination()
	if err != nil {
		return nil, err
	}

	orderByList, err = parser.ParseOrderBy(in.OrderBy)
	if err != nil {
		return nil, err
	}
	var sortQuery *dataselector.SortQuery
	if len(orderByList) == 0 {
		sortQuery = dataselector.NoSort
	} else {
		sortQuery = dataselector.NewSortQueryOrderBy(orderByList)
	}

	filterByList, err := parser.ParseFilterBy(in.Filter)
	if err != nil {
		return nil, err
	}
	var filterQuery *dataselector.FilterQuery
	if len(filterByList) == 0 {
		filterQuery = dataselector.NoFilter
	} else {
		filterQuery = dataselector.NewFilterQueryFilterBy(filterByList)
	}

	ds := dataselector.DataSelector{
		GenericDataList: datatypes.ToDataItemsFromDeployments(deployments),
		DataSelectQuery: &dataselector.DataSelectQuery{
			PaginationQuery: paginationQuery,
			SortQuery:       sortQuery,
			FilterQuery:     filterQuery,
		},
	}

	selectedDeployments := datatypes.FromDataItemsToDeployments(ds.Filter().Sort().Paginate().GenericDataList)
	return selectedDeployments, nil

}

func selectClusters(in *deploymentpb.ListClustersRequest, clusterInfoList []*deploymentpb.ClusterInfo) ([]*deploymentpb.ClusterInfo, error) {
	var orderByList []dataselector.OrderBy

	pageSize, offset := convertPaginationParams(in.PageSize, in.Offset)
	paginationQuery := newPaginationQuery(pageSize, offset)

	err := paginationQuery.IsValidPagination()
	if err != nil {
		return nil, err
	}

	orderByList, err = parser.ParseOrderBy(in.OrderBy)
	if err != nil {
		return nil, err
	}
	var sortQuery *dataselector.SortQuery
	if len(orderByList) == 0 {
		sortQuery = dataselector.NoSort
	} else {
		sortQuery = dataselector.NewSortQueryOrderBy(orderByList)
	}

	filterByList, err := parser.ParseFilterBy(in.Filter)
	if err != nil {
		return nil, err
	}
	var filterQuery *dataselector.FilterQuery
	if len(filterByList) == 0 {
		filterQuery = dataselector.NoFilter
	} else {
		filterQuery = dataselector.NewFilterQueryFilterBy(filterByList)
	}

	ds := dataselector.DataSelector{
		GenericDataList: datatypes.ToDataItemsFromClusterInfoList(clusterInfoList),
		DataSelectQuery: &dataselector.DataSelectQuery{
			PaginationQuery: paginationQuery,
			SortQuery:       sortQuery,
			FilterQuery:     filterQuery,
		},
	}
	selectedClusters := datatypes.FromDataItemsToClusterInfoList(ds.Filter().Sort().Paginate().GenericDataList)
	return selectedClusters, nil
}

func selectClustersPerDeployment(in *deploymentpb.ListDeploymentClustersRequest, clusterList []*deploymentpb.Cluster) ([]*deploymentpb.Cluster, error) {
	var orderByList []dataselector.OrderBy

	pageSize, offset := convertPaginationParams(in.PageSize, in.Offset)
	paginationQuery := newPaginationQuery(pageSize, offset)

	err := paginationQuery.IsValidPagination()
	if err != nil {
		return nil, err
	}

	orderByList, err = parser.ParseOrderBy(in.OrderBy)
	if err != nil {
		return nil, err
	}
	var sortQuery *dataselector.SortQuery
	if len(orderByList) == 0 {
		sortQuery = dataselector.NoSort
	} else {
		sortQuery = dataselector.NewSortQueryOrderBy(orderByList)
	}

	filterByList, err := parser.ParseFilterBy(in.Filter)
	if err != nil {
		return nil, err
	}
	var filterQuery *dataselector.FilterQuery
	if len(filterByList) == 0 {
		filterQuery = dataselector.NoFilter
	} else {
		filterQuery = dataselector.NewFilterQueryFilterBy(filterByList)
	}

	ds := dataselector.DataSelector{
		GenericDataList: datatypes.ToDataItemsFromClusterList(clusterList),
		DataSelectQuery: &dataselector.DataSelectQuery{
			PaginationQuery: paginationQuery,
			SortQuery:       sortQuery,
			FilterQuery:     filterQuery,
		},
	}

	selectedClusters := datatypes.FromDataItemsToClusterList(ds.Filter().Sort().Paginate().GenericDataList)
	return selectedClusters, nil

}

func newPaginationQuery(pageSize uint32, offset uint32) *dataselector.PaginationQuery {
	if pageSize == 0 && offset == 0 {
		return dataselector.NewPaginationQuery(dataselector.DefaultPageSize, 0)
	}

	if pageSize == 0 && offset != 0 {
		return dataselector.NewPaginationQuery(dataselector.DefaultPageSize, int(offset))
	}

	return dataselector.NewPaginationQuery(int(pageSize), int(offset))
}
