// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package datatypes

import (
	deploymentpb "github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/api/nbi/v2/deployment/v1"
	"github.com/open-edge-platform/app-orch-deployment/app-deployment-manager/pkg/utils/dataselector"
	"reflect"
	"testing"
)

func getOrderByDeploymentID(deploymentList []*deploymentpb.Deployment) []string {
	idOrder := []string{}
	for _, e := range deploymentList {
		idOrder = append(idOrder, e.DeployId)
	}
	return idOrder
}

type PaginationAndSortTestCase struct {
	Info            string
	SortQuery       *dataselector.SortQuery
	PaginationQuery *dataselector.PaginationQuery
	ExpectedOrder   []string
}

type PaginationSortFilterTestCase struct {
	Info            string
	SortQuery       *dataselector.SortQuery
	PaginationQuery *dataselector.PaginationQuery
	FilterQuery     *dataselector.FilterQuery
	ExpectedOrder   []string
}

type SortTestCase struct {
	Info          string
	SortQuery     *dataselector.SortQuery
	ExpectedOrder []string
}

type PaginationTestCase struct {
	Info            string
	PaginationQuery *dataselector.PaginationQuery
	ExpectedOrder   []string
}

type FilterTestCase struct {
	Info              string
	FilterQuery       *dataselector.FilterQuery
	ExpectedDataItems []*deploymentpb.Deployment
}

func getDeploymentList() []dataselector.DataItem {
	return ToDataItemsFromDeployments([]*deploymentpb.Deployment{
		{
			Name:        "dep-b",
			DeployId:    "2",
			DisplayName: "deployment-2",
			AppName:     "test-app-2",
			AppVersion:  "1.0.1",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_RUNNING,
			},
		},
		{
			Name:        "dep-a",
			DeployId:    "1",
			DisplayName: "deployment-1",
			AppName:     "test-app-1",
			AppVersion:  "1.2.1",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_RUNNING,
			},
		},
		{
			Name:        "dep-c",
			DeployId:    "3",
			DisplayName: "deployment-3",
			AppName:     "test-app-3",
			AppVersion:  "1.2.2",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_DOWN,
			},
		},
		{
			Name:        "dep-e",
			DeployId:    "5",
			DisplayName: "deployment-5",
			AppName:     "test-app-5",
			AppVersion:  "1.0.0",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_TERMINATING,
			},
		},
		{
			Name:        "dep-d",
			DeployId:    "4",
			DisplayName: "deployment-4",
			AppName:     "test-app-4",
			AppVersion:  "1.1.1",
			Status: &deploymentpb.Deployment_Status{
				State: deploymentpb.State_UPDATING,
			},
		},
	})
}

func TestSortDeployments(t *testing.T) {
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
			"sort by name/asc and status/desc",
			dataselector.NewSortQuery([]string{dataselector.ASC, "name", dataselector.DECS, "status"}),
			[]string{"1", "2", "3", "4", "5"},
		},
		{
			"sort by  name/asc and app version/desc",
			dataselector.NewSortQuery([]string{dataselector.ASC, "name", dataselector.DECS, "appVersion"}),
			[]string{"1", "2", "3", "4", "5"},
		},
		{
			"sort by name/desc and app version/asc",
			dataselector.NewSortQuery([]string{dataselector.DECS, "name", dataselector.ASC, "appVersion"}),
			[]string{"5", "4", "3", "2", "1"},
		},
		{
			"sort based on app version/asc",
			dataselector.NewSortQuery([]string{dataselector.ASC, "appVersion"}),
			[]string{"5", "2", "4", "1", "3"},
		},
		{
			"sort by status/asc",
			dataselector.NewSortQuery([]string{dataselector.ASC, "status"}),
			[]string{"3", "2", "1", "5", "4"},
		},
		{
			"sort by appVersion/asc",
			dataselector.NewSortQuery([]string{dataselector.ASC, "appVersion"}),
			[]string{"5", "2", "4", "1", "3"},
		},
	}
	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getDeploymentList(),
			DataSelectQuery: &dataselector.DataSelectQuery{SortQuery: testCase.SortQuery},
		}
		sortedData := FromDataItemsToDeployments(ds.Sort().GenericDataList)
		order := getOrderByDeploymentID(sortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Sort: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.SortQuery, order, testCase.ExpectedOrder)
		}
	}

}

func TestPagination(t *testing.T) {
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
			GenericDataList: getDeploymentList(),
			DataSelectQuery: &dataselector.DataSelectQuery{PaginationQuery: testCase.PaginationQuery},
		}
		paginatedData := FromDataItemsToDeployments(ds.Paginate().GenericDataList)
		order := getOrderByDeploymentID(paginatedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}

}

func TestPaginationAndSort(t *testing.T) {
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
			Info:            "request 4 items from existing page and sort dsc by deploy id",
			PaginationQuery: dataselector.NewPaginationQuery(4, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.DECS, "deployId"}),
			ExpectedOrder:   []string{"5", "3", "2", "1"},
		},
		{
			Info:            "request 5 items from existing page and sort asc by status",
			PaginationQuery: dataselector.NewPaginationQuery(5, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.ASC, "status"}),
			ExpectedOrder:   []string{"3", "2", "1", "5", "4"},
		},
	}
	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getDeploymentList(),
			DataSelectQuery: &dataselector.DataSelectQuery{
				PaginationQuery: testCase.PaginationQuery,
				SortQuery:       testCase.SortQuery,
			},
		}
		paginatedAndSortedData := FromDataItemsToDeployments(ds.Paginate().Sort().GenericDataList)
		order := getOrderByDeploymentID(paginatedAndSortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}

}

func TestFilter(t *testing.T) {
	testCases := []FilterTestCase{
		{
			Info: "Filter by name and wildcard at the beginning",
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("*-a"),
				},
			}),
			ExpectedDataItems: []*deploymentpb.Deployment{
				{
					Name:        "dep-a",
					DeployId:    "1",
					DisplayName: "deployment-1",
					AppName:     "test-app-1",
					AppVersion:  "1.2.1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
			},
		},
		{
			Info: "Filter by name and wildcard at the end",
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("dep*"),
				},
			}),
			ExpectedDataItems: []*deploymentpb.Deployment{
				{
					Name:        "dep-b",
					DeployId:    "2",
					DisplayName: "deployment-2",
					AppName:     "test-app-2",
					AppVersion:  "1.0.1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
				{
					Name:        "dep-a",
					DeployId:    "1",
					DisplayName: "deployment-1",
					AppName:     "test-app-1",
					AppVersion:  "1.2.1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
				{
					Name:        "dep-c",
					DeployId:    "3",
					DisplayName: "deployment-3",
					AppName:     "test-app-3",
					AppVersion:  "1.2.2",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_DOWN,
					},
				},
				{
					Name:        "dep-e",
					DeployId:    "5",
					DisplayName: "deployment-5",
					AppName:     "test-app-5",
					AppVersion:  "1.0.0",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_TERMINATING,
					},
				},
				{
					Name:        "dep-d",
					DeployId:    "4",
					DisplayName: "deployment-4",
					AppName:     "test-app-4",
					AppVersion:  "1.1.1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_UPDATING,
					},
				},
			},
		},
		{
			Info: "Filter by appVersion",
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "appVersion",
					Value: dataselector.StdComparableString("1.2.*"),
				},
			}),
			ExpectedDataItems: []*deploymentpb.Deployment{
				{
					Name:        "dep-a",
					DeployId:    "1",
					DisplayName: "deployment-1",
					AppName:     "test-app-1",
					AppVersion:  "1.2.1",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_RUNNING,
					},
				},
				{
					Name:        "dep-c",
					DeployId:    "3",
					DisplayName: "deployment-3",
					AppName:     "test-app-3",
					AppVersion:  "1.2.2",
					Status: &deploymentpb.Deployment_Status{
						State: deploymentpb.State_DOWN,
					},
				},
			},
		},
	}

	for _, testCase := range testCases {
		selectableData := dataselector.DataSelector{
			GenericDataList: getDeploymentList(),
			DataSelectQuery: &dataselector.DataSelectQuery{FilterQuery: testCase.FilterQuery},
		}
		filteredData := FromDataItemsToDeployments(selectableData.Filter().GenericDataList)
		if !reflect.DeepEqual(filteredData, testCase.ExpectedDataItems) {
			t.Errorf(`Filtering: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.FilterQuery, filteredData, testCase.ExpectedDataItems)
		}
	}
}

func TestPaginationSortFilter(t *testing.T) {
	testCases := []PaginationSortFilterTestCase{
		{
			Info:            "request 2 item from existing page and no sort and filter using appVersion",
			PaginationQuery: dataselector.NewPaginationQuery(2, 0),
			SortQuery:       dataselector.NoSort,
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "appVersion",
					Value: dataselector.StdComparableString("1.2.*"),
				},
			}),
			ExpectedOrder: []string{"1"},
		},
		{
			Info:            "request 4 items from existing page and sort dsc by deploy id and filter using name",
			PaginationQuery: dataselector.NewPaginationQuery(4, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.DECS, "deployId"}),
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "name",
					Value: dataselector.StdComparableString("*-e"),
				},
				{
					Field: "name",
					Value: dataselector.StdComparableString("*-c"),
				},
			}),
			ExpectedOrder: []string{"5", "3"},
		},
		{
			Info:            "request 5 items from existing page and sort asc by status and filter using appVersion",
			PaginationQuery: dataselector.NewPaginationQuery(5, 0),
			SortQuery:       dataselector.NewSortQuery([]string{dataselector.ASC, "status"}),
			FilterQuery: dataselector.NewFilterQueryFilterBy([]dataselector.FilterBy{
				{
					Field: "appVersion",
					Value: dataselector.StdComparableString("1.0.*"),
				},
			}),
			ExpectedOrder: []string{"2", "5"},
		},
	}
	for _, testCase := range testCases {
		ds := dataselector.DataSelector{
			GenericDataList: getDeploymentList(),
			DataSelectQuery: &dataselector.DataSelectQuery{
				PaginationQuery: testCase.PaginationQuery,
				SortQuery:       testCase.SortQuery,
				FilterQuery:     testCase.FilterQuery,
			},
		}
		paginatedAndSortedData := FromDataItemsToDeployments(ds.Paginate().Sort().Filter().GenericDataList)
		order := getOrderByDeploymentID(paginatedAndSortedData)
		if !reflect.DeepEqual(order, testCase.ExpectedOrder) {
			t.Errorf(`Pagination/Sort/Filter: %s. Received invalid items for %+v. Got %v, expected %v.`,
				testCase.Info, testCase.PaginationQuery, order, testCase.ExpectedOrder)
		}
	}
}
