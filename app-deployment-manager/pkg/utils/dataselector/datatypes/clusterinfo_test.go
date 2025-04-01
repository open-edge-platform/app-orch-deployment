// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
	"reflect"
	"testing"
)

func getOrderByClusterID(clusterInfoList []*deploymentpb.ClusterInfo) []string {
	idOrder := []string{}
	for _, e := range clusterInfoList {
		idOrder = append(idOrder, e.Id)
	}
	return idOrder
}

func getClusterInfoList() []dataselector.DataItem {
	return ToDataItemsFromClusterInfoList([]*deploymentpb.ClusterInfo{
		{
			Name: "cluster-b",
			Id:   "2",
		},
		{
			Name: "cluster-a",
			Id:   "1",
		},
		{
			Name: "cluster-c",
			Id:   "3",
		},
		{
			Name: "cluster-e",
			Id:   "5",
		},
		{
			Name: "cluster-d",
			Id:   "4",
		},
	})
}

func TestSortClusterInfoList(t *testing.T) {
	testCases := []SortTestCase{
		{
			"no sort - do not change the original order",
			dataselector.NoSort,
			[]string{"2", "1", "3", "5", "4"},
		},
		{
			"ascending sort by name",
			dataselector.NewSortQuery([]string{dataselector.ASC, "name"}),
			[]string{"1", "2", "3", "4", "5"},
		},
		{
			"descending sort by name",
			dataselector.NewSortQuery([]string{dataselector.DECS, "name"}),
			[]string{"5", "4", "3", "2", "1"},
		},
		{
			"sort by name/asc and id/asc",
			dataselector.NewSortQuery([]string{dataselector.ASC, "name", dataselector.ASC, "id"}),
			[]string{"1", "2", "3", "4", "5"},
		},
		{
			"descending sort by id",
			dataselector.NewSortQuery([]string{dataselector.DECS, "id"}),
			[]string{"5", "4", "3", "2", "1"},
		},
	}

	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getClusterInfoList(),
			DataSelectQuery: &dataselector.DataSelectQuery{SortQuery: testCase.SortQuery},
		}
		sortedData := FromDataItemsToClusterInfoList(ds.Sort().GenericDataList)
		order := getOrderByClusterID(sortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Sort: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.SortQuery, order, testCase.ExpectedOrder)
		}
	}
}

func TestPaginationClusterInfo(t *testing.T) {
	testCases := []PaginationTestCase{

		{
			"request one item from existing page - element should be returned",
			dataselector.NewPaginationQuery(1, 4),
			[]string{"4"},
		},
		{
			"request 2 items from existing page - 2 elements should be returned",
			dataselector.NewPaginationQuery(2, 0),
			[]string{"2", "1"},
		},
		{
			"request 2 items from existing page - 1 element should be returned because last page has 1 element",
			dataselector.NewPaginationQuery(2, 4),
			[]string{"4"},
		},
		{
			"request 3 items from non existing page - no elements should be returned",
			dataselector.NewPaginationQuery(3, 15),
			[]string{},
		},
		{
			"empty pagination - no elements should be returned",
			dataselector.EmptyPagination,
			[]string{},
		},
		{
			"request 4 items from existing page",
			dataselector.NewPaginationQuery(4, 0),
			[]string{"2", "1", "3", "5"},
		},
	}

	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getClusterInfoList(),
			DataSelectQuery: &dataselector.DataSelectQuery{PaginationQuery: testCase.PaginationQuery},
		}
		paginatedData := FromDataItemsToClusterInfoList(ds.Paginate().GenericDataList)
		order := getOrderByClusterID(paginatedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}
}

func TestPaginationAndSortClusterInfo(t *testing.T) {
	testCases := []PaginationAndSortTestCase{
		{
			Info:            "request 2 item from existing page and no sort",
			PaginationQuery: dataselector.NewPaginationQuery(2, 0),
			SortQuery:       dataselector.NoSort,
			ExpectedOrder:   []string{"2", "1"},
		},
		{
			Info:            "request 2 item from existing page and sort asc by name",
			PaginationQuery: dataselector.NewPaginationQuery(2, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.ASC, "name"}),
			ExpectedOrder:   []string{"1", "2"},
		},
		{
			Info:            "request 4 items from existing page and sort dsc by id",
			PaginationQuery: dataselector.NewPaginationQuery(4, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.DECS, "id"}),
			ExpectedOrder:   []string{"5", "3", "2", "1"},
		},
		{
			Info:            "request 5 items from existing page and sort asc by name",
			PaginationQuery: dataselector.NewPaginationQuery(5, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.DECS, "name"}),
			ExpectedOrder:   []string{"5", "4", "3", "2", "1"},
		},
	}
	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getClusterInfoList(),
			DataSelectQuery: &dataselector.DataSelectQuery{
				PaginationQuery: testCase.PaginationQuery,
				SortQuery:       testCase.SortQuery,
			},
		}
		paginatedAndSortedData := FromDataItemsToClusterInfoList(ds.Paginate().Sort().GenericDataList)
		order := getOrderByClusterID(paginatedAndSortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}
}
