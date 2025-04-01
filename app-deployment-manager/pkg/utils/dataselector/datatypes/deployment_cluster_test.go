// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
	"reflect"
	"testing"
)

type FilterDeploymentClusterTestCase struct {
	Info              string
	FilterQuery       *dataselector.FilterQuery
	ExpectedDataItems []*deploymentpb.Cluster
}

func getDeploymentClusterList() []dataselector.DataItem {
	return ToDataItemsFromClusterList([]*deploymentpb.Cluster{
		{
			Name: "cluster-1",
			Id:   "cluster-1",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_RUNNING,
			},
		},
		{
			Name: "cluster-2",
			Id:   "cluster-2",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_RUNNING,
			},
		},
		{
			Name: "cluster-3",
			Id:   "cluster-3",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_DOWN,
			},
		},
		{
			Name: "cluster-4",
			Id:   "cluster-4",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_TERMINATING,
			},
		},
		{
			Name: "cluster-5",
			Id:   "cluster-5",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_UPDATING,
			},
		},
	})
}

func TestFilterDeploymentCluster(t *testing.T) {
	testCases := []FilterDeploymentClusterTestCase{
		{
			Info: "Filter by name and wildcard at the beginning",
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("*-1"),
				},
			}),
			ExpectedDataItems: []*deploymentpb.Cluster{
				{
					Name: "cluster-1",
					Id:   "cluster-1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
			},
		},
		{
			Info: "Filter by status-RUNNING",
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "status",
					Value: dataselector.StdComparableString("RUNNING"),
				},
			}),
			ExpectedDataItems: []*deploymentpb.Cluster{
				{
					Name: "cluster-1",
					Id:   "cluster-1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
				{
					Name: "cluster-2",
					Id:   "cluster-2",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
			},
		},
		{
			Info: "Filter by status-DOWN",
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "status",
					Value: dataselector.StdComparableString("DOWN"),
				},
			}),
			ExpectedDataItems: []*deploymentpb.Cluster{
				{
					Name: "cluster-3",
					Id:   "cluster-3",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_DOWN,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		selectableData := dataselector.DataSelector{
			GenericDataList: getDeploymentClusterList(),
			DataSelectQuery: &dataselector.DataSelectQuery{FilterQuery: testCase.FilterQuery},
		}
		filteredData := FromDataItemsToClusterList(selectableData.Filter().GenericDataList)
		if !reflect.DeepEqual(filteredData, testCase.ExpectedDataItems) {
			t.Errorf(`Filtering: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.FilterQuery, filteredData, testCase.ExpectedDataItems)
		}
	}
}

func TestPaginationSortFilterDeploymentCluster(t *testing.T) {
	testCases := []PaginationSortFilterTestCase{
		{
			Info:            "request 2 item from existing page and no sort and filter using name",
			PaginationQuery: dataselector.NewPaginationQuery(2, 0),
			SortQuery:       dataselector.NoSort,
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("cluster-1"),
				},
			}),
			ExpectedOrder: []string{"cluster-1"},
		},
		{
			Info:            "request 4 items from existing page and sort dsc by deploy id and filter using name and id",
			PaginationQuery: dataselector.NewPaginationQuery(4, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.DECS, "name"}),
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("*-1"),
				},
				{
					Field: "id",
					Value: dataselector.StdComparableString("*-2"),
				},
			}),
			ExpectedOrder: []string{"cluster-2", "cluster-1"},
		},
	}
	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getDeploymentClusterList(),
			DataSelectQuery: &dataselector.DataSelectQuery{
				PaginationQuery: testCase.PaginationQuery,
				SortQuery:       testCase.SortQuery,
				FilterQuery:     testCase.FilterQuery,
			},
		}
		paginatedAndSortedData := FromDataItemsToClusterList(ds.Paginate().Sort().Filter().GenericDataList)
		order := getOrderByDeploymentClusterID(paginatedAndSortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination/Sort/Filter: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}
}

func getOrderByDeploymentClusterID(deploymentClusterList []*deploymentpb.Cluster) []string {
	idOrder := []string{}
	for _, e := range deploymentClusterList {
		idOrder = append(idOrder, e.Id)
	}
	return idOrder
}
